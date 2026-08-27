package parser

import (
	"fmt"
	"strings"
)

// ApplyAWSTopology inspects node attributes and constructs the AWS architectural hierarchy:
// VPC -> Availability Zone -> Subnet -> Resource
func ApplyAWSTopology(nodeMap map[string]Node, edgeMap map[string]Edge) (map[string]Node, map[string]Edge) {
	if nodeMap == nil {
		return nodeMap, edgeMap
	}

	// 1. Build lookup tables for VPCs and Subnets
	vpcLookup := make(map[string]string)    // raw reference / id -> canonical VPC node ID
	subnetLookup := make(map[string]string) // raw reference / id -> canonical Subnet node ID
	var defaultVPCID string

	for id, n := range nodeMap {
		resType := n.Data.ResourceType
		if resType == "aws_vpc" || resType == "azurerm_virtual_network" || resType == "google_compute_network" {
			vpcLookup[id] = id
			vpcLookup[n.Data.Label] = id
			if shortID := getShortID(id); shortID != "" {
				vpcLookup[shortID] = id
			}
			if attrID, ok := n.Data.Attributes["id"].(string); ok && attrID != "" {
				vpcLookup[attrID] = id
			}
			if defaultVPCID == "" {
				defaultVPCID = id
			}
		} else if resType == "aws_subnet" || resType == "azurerm_subnet" {
			subnetLookup[id] = id
			subnetLookup[n.Data.Label] = id
			if shortID := getShortID(id); shortID != "" {
				subnetLookup[shortID] = id
			}
			if attrID, ok := n.Data.Attributes["id"].(string); ok && attrID != "" {
				subnetLookup[attrID] = id
			}
		}
	}

	// 2. Process Subnets: Identify Availability Zones & link Subnets -> AZs -> VPCs
	for id, n := range nodeMap {
		if n.Data.ResourceType != "aws_subnet" {
			continue
		}

		attrs := n.Data.Attributes
		if attrs == nil {
			attrs = make(map[string]interface{})
		}

		// Determine parent VPC for this subnet
		subnetVPCID := ""
		if vpcIDVal, ok := attrs["vpc_id"].(string); ok && vpcIDVal != "" {
			subnetVPCID = resolveTargetFromLookup(vpcIDVal, vpcLookup)
		}
		if subnetVPCID == "" {
			subnetVPCID = defaultVPCID
		}

		// Check if Subnet declares an availability_zone
		azRaw, hasAZ := attrs["availability_zone"].(string)
		if !hasAZ || azRaw == "" {
			// Fallback: check if subnet label contains az suffix (e.g. public_1a -> us-east-1a or a)
			azRaw = inferAZFromLabel(n.Data.Label)
		}

		// Public vs Private classification
		if isPublicSubnet(n.Data.Label, attrs) {
			attrs["subnet_type"] = "public"
		} else {
			attrs["subnet_type"] = "private"
		}
		n.Data.Attributes = attrs

		if azRaw != "" {
			cleanAZ := cleanAZName(azRaw)
			azNodeID := fmt.Sprintf("az-%s", cleanAZ)

			// Create synthetic AZ Container Node if not already present
			if _, exists := nodeMap[azNodeID]; !exists {
				nodeMap[azNodeID] = Node{
					ID:   azNodeID,
					Type: "infrastructureNode",
					Position: Position{
						X: 0,
						Y: 0,
					},
					Data: NodeData{
						Label:        fmt.Sprintf("AZ: %s", cleanAZ),
						Provider:     "aws",
						ResourceType: "aws_availability_zone",
						Module:       n.Data.Module,
						IsContainer:  true,
						DriftStatus:  "IN_SYNC",
						Attributes: map[string]interface{}{
							"availability_zone": cleanAZ,
							"vpc_id":            subnetVPCID,
						},
					},
					ParentID: subnetVPCID, // AZ is linked directly to VPC
				}
			}

			// Subnet is linked directly to the synthetic AZ container
			n.ParentID = azNodeID
		} else if subnetVPCID != "" {
			// Subnet has no AZ specified -> attach directly to VPC
			n.ParentID = subnetVPCID
		}

		nodeMap[id] = n
	}

	// 3. Process Compute / Database / Container / Networking Resources -> Link to Subnets or VPC
	for id, n := range nodeMap {
		resType := n.Data.ResourceType
		// Skip container infrastructure itself
		if resType == "aws_vpc" || resType == "aws_availability_zone" || resType == "aws_subnet" || resType == "module" {
			continue
		}

		attrs := n.Data.Attributes
		if attrs == nil {
			continue
		}

		matchedSubnetID := ""

		// Check subnet_id (e.g., aws_instance, aws_db_instance, aws_nat_gateway)
		if subnetIDVal, ok := attrs["subnet_id"].(string); ok && subnetIDVal != "" {
			matchedSubnetID = resolveTargetFromLookup(subnetIDVal, subnetLookup)
		}

		// Check subnet_ids (e.g., aws_eks_cluster, aws_lb, aws_db_subnet_group)
		if matchedSubnetID == "" {
			if subnetsVal, ok := attrs["subnet_ids"]; ok {
				matchedSubnetID = extractFirstSubnetMatch(subnetsVal, subnetLookup)
			}
		}

		if matchedSubnetID != "" {
			n.ParentID = matchedSubnetID
			nodeMap[id] = n
			continue
		}

		// If resource has no specific subnet but has vpc_id (e.g. Internet Gateway, Route Table, Security Group, ALB)
		if n.ParentID == "" || strings.HasPrefix(n.ParentID, "module.") {
			if vpcIDVal, ok := attrs["vpc_id"].(string); ok && vpcIDVal != "" {
				matchedVPCID := resolveTargetFromLookup(vpcIDVal, vpcLookup)
				if matchedVPCID != "" {
					n.ParentID = matchedVPCID
					nodeMap[id] = n
				}
			}
		}
	}

	// 4. Update IsContainer dynamically based on final ParentID graph
	parentSet := make(map[string]bool)
	for _, n := range nodeMap {
		if n.ParentID != "" {
			parentSet[n.ParentID] = true
		}
	}

	for id, n := range nodeMap {
		resType := n.Data.ResourceType
		if resType == "aws_vpc" || resType == "aws_availability_zone" || resType == "aws_subnet" || resType == "module" || resType == "azurerm_virtual_network" || resType == "google_compute_network" {
			n.Data.IsContainer = parentSet[id]
			nodeMap[id] = n
		}
	}

	return nodeMap, edgeMap
}

func resolveTargetFromLookup(val string, lookup map[string]string) string {
	val = strings.TrimSpace(val)
	if match, exists := lookup[val]; exists {
		return match
	}

	// Handle interpolations / attribute references like aws_vpc.main.id -> aws_vpc.main
	if strings.Contains(val, ".") {
		parts := strings.Split(val, ".")
		for i := 0; i < len(parts)-1; i++ {
			candidate := fmt.Sprintf("%s.%s", parts[i], parts[i+1])
			if match, exists := lookup[candidate]; exists {
				return match
			}
		}
	}

	// Remove index e.g. aws_subnet.public[0].id -> aws_subnet.public
	cleaned := cleanResourceReference(val)
	if match, exists := lookup[cleaned]; exists {
		return match
	}

	return ""
}

func extractFirstSubnetMatch(subnetsVal interface{}, lookup map[string]string) string {
	switch v := subnetsVal.(type) {
	case []interface{}:
		for _, elem := range v {
			if strVal, ok := elem.(string); ok {
				if match := resolveTargetFromLookup(strVal, lookup); match != "" {
					return match
				}
			}
		}
	case string:
		if match := resolveTargetFromLookup(v, lookup); match != "" {
			return match
		}
		// Handle split by comma
		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			for _, part := range parts {
				if match := resolveTargetFromLookup(part, lookup); match != "" {
					return match
				}
			}
		}
	}
	return ""
}

func cleanAZName(az string) string {
	az = strings.TrimSpace(az)
	// If interpolation like ${var.region}a or var.az_a
	az = strings.TrimPrefix(az, "${")
	az = strings.TrimSuffix(az, "}")
	if strings.Contains(az, ".") {
		parts := strings.Split(az, ".")
		az = parts[len(parts)-1]
	}
	return az
}

func inferAZFromLabel(label string) string {
	l := strings.ToLower(label)
	for _, suffix := range []string{"1a", "1b", "1c", "2a", "2b", "2c", "us-east-1a", "us-east-1b", "us-west-2a", "us-west-2b"} {
		if strings.Contains(l, suffix) {
			return suffix
		}
	}
	return ""
}

func isPublicSubnet(label string, attrs map[string]interface{}) bool {
	l := strings.ToLower(label)
	if strings.Contains(l, "public") {
		return true
	}
	if mapPub, ok := attrs["map_public_ip_on_launch"].(bool); ok && mapPub {
		return true
	}
	if tags, ok := attrs["tags"].(map[string]interface{}); ok {
		for k, v := range tags {
			if strings.ToLower(k) == "type" && strings.ToLower(fmt.Sprintf("%v", v)) == "public" {
				return true
			}
		}
	}
	return false
}

func getShortID(fullID string) string {
	parts := strings.Split(fullID, ".")
	if len(parts) >= 2 {
		return fmt.Sprintf("%s.%s", parts[len(parts)-2], parts[len(parts)-1])
	}
	return fullID
}
