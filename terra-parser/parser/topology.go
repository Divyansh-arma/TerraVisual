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

	// 0. Perform Semantic Module Decomposition for standard AWS modules (VPC, EKS, S3, DynamoDB)
	nodeMap, edgeMap = DecomposeSemanticModules(nodeMap, edgeMap)

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

		// If resource has no specific subnet and is not already nested in a module, but has vpc_id (e.g. Internet Gateway, Route Table, Security Group, ALB)
		if n.ParentID == "" {
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

// ExpandSemanticAWSModules unpacks official AWS modules (e.g. terraform-aws-modules/vpc/aws, terraform-aws-modules/eks/aws)
// into their semantic child resources when remote definitions don't provide child .tf files.
func ExpandSemanticAWSModules(nodeMap map[string]Node, edgeMap map[string]Edge) (map[string]Node, map[string]Edge) {
	// Find any VPC node already created
	var defaultVPCNodeID string
	for id, n := range nodeMap {
		if n.Data.ResourceType == "aws_vpc" {
			defaultVPCNodeID = id
			break
		}
	}

	createdSubnets := make([]string, 0)
	createdPrivateSubnets := make([]string, 0)

	for id, n := range nodeMap {
		if n.Data.ResourceType != "module" {
			continue
		}

		attrs := n.Data.Attributes
		if attrs == nil {
			continue
		}

		src, _ := attrs["source"].(string)
		modName := strings.TrimPrefix(id, "module.")

		// 1. Expand terraform-aws-modules/vpc/aws
		if strings.Contains(src, "aws-modules/vpc") || strings.Contains(src, "terraform-aws-modules/vpc") {
			// Check if child VPC already exists
			hasChildVPC := false
			for _, child := range nodeMap {
				if child.ParentID == id && child.Data.ResourceType == "aws_vpc" {
					hasChildVPC = true
					break
				}
			}

			vpcID := fmt.Sprintf("module.%s.aws_vpc.this", modName)
			if !hasChildVPC {
				cidr := "10.0.0.0/16"
				if c, ok := attrs["cidr"].(string); ok && c != "" {
					cidr = c
				} else if c, ok := attrs["cidr_block"].(string); ok && c != "" {
					cidr = c
				}

				nodeMap[vpcID] = Node{
					ID:   vpcID,
					Type: "infrastructureNode",
					Position: Position{
						X: 0,
						Y: 0,
					},
					Data: NodeData{
						Label:        fmt.Sprintf("%s-vpc", modName),
						Provider:     "aws",
						ResourceType: "aws_vpc",
						Module:       id,
						IsContainer:  true,
						DriftStatus:  "IN_SYNC",
						Attributes: map[string]interface{}{
							"cidr_block": cidr,
						},
					},
					ParentID: id,
				}
				defaultVPCNodeID = vpcID
			} else {
				vpcID = defaultVPCNodeID
			}

			// Extract AZs
			azs := extractStringList(attrs["azs"])
			if len(azs) == 0 {
				azs = []string{"us-east-1a", "us-east-1b", "us-east-1c"}
			}

			// Generate Public Subnets
			publicSubnets := extractStringList(attrs["public_subnets"])
			for i, subCIDR := range publicSubnets {
				subID := fmt.Sprintf("module.%s.aws_subnet.public_%d", modName, i+1)
				az := azs[i%len(azs)]
				if _, exists := nodeMap[subID]; !exists {
					nodeMap[subID] = Node{
						ID:   subID,
						Type: "infrastructureNode",
						Position: Position{
							X: 0,
							Y: 0,
						},
						Data: NodeData{
							Label:        fmt.Sprintf("%s-public-%d", modName, i+1),
							Provider:     "aws",
							ResourceType: "aws_subnet",
							Module:       id,
							IsContainer:  false,
							DriftStatus:  "IN_SYNC",
							Attributes: map[string]interface{}{
								"cidr_block":        subCIDR,
								"subnet_type":       "public",
								"availability_zone": az,
								"vpc_id":            vpcID,
							},
						},
						ParentID: vpcID,
					}
					createdSubnets = append(createdSubnets, subID)
				}
			}

			// Generate Private Subnets
			privateSubnets := extractStringList(attrs["private_subnets"])
			for i, subCIDR := range privateSubnets {
				subID := fmt.Sprintf("module.%s.aws_subnet.private_%d", modName, i+1)
				az := azs[i%len(azs)]
				if _, exists := nodeMap[subID]; !exists {
					nodeMap[subID] = Node{
						ID:   subID,
						Type: "infrastructureNode",
						Position: Position{
							X: 0,
							Y: 0,
						},
						Data: NodeData{
							Label:        fmt.Sprintf("%s-private-%d", modName, i+1),
							Provider:     "aws",
							ResourceType: "aws_subnet",
							Module:       id,
							IsContainer:  false,
							DriftStatus:  "IN_SYNC",
							Attributes: map[string]interface{}{
								"cidr_block":        subCIDR,
								"subnet_type":       "private",
								"availability_zone": az,
								"vpc_id":            vpcID,
							},
						},
						ParentID: vpcID,
					}
					createdSubnets = append(createdSubnets, subID)
					createdPrivateSubnets = append(createdPrivateSubnets, subID)
				}
			}

			n.Data.IsContainer = true
			nodeMap[id] = n
		}

		// 2. Expand terraform-aws-modules/eks/aws
		if strings.Contains(src, "aws-modules/eks") || strings.Contains(src, "terraform-aws-modules/eks") {
			clusterName := fmt.Sprintf("%s-cluster", modName)
			if cn, ok := attrs["cluster_name"].(string); ok && cn != "" {
				clusterName = cn
			}

			clusterID := fmt.Sprintf("module.%s.aws_eks_cluster.this", modName)
			ngID := fmt.Sprintf("module.%s.aws_eks_node_group.this", modName)

			// Determine parent container for EKS (place inside VPC container if available, or inside module)
			targetParent := id
			if defaultVPCNodeID != "" {
				targetParent = defaultVPCNodeID
			}

			if _, exists := nodeMap[clusterID]; !exists {
				nodeMap[clusterID] = Node{
					ID:   clusterID,
					Type: "infrastructureNode",
					Position: Position{
						X: 0,
						Y: 0,
					},
					Data: NodeData{
						Label:        clusterName,
						Provider:     "aws",
						ResourceType: "aws_eks_cluster",
						Module:       id,
						IsContainer:  false,
						DriftStatus:  "IN_SYNC",
						Attributes: map[string]interface{}{
							"cluster_name": clusterName,
							"vpc_id":       defaultVPCNodeID,
						},
					},
					ParentID: targetParent,
				}
			}

			if _, exists := nodeMap[ngID]; !exists {
				nodeMap[ngID] = Node{
					ID:   ngID,
					Type: "infrastructureNode",
					Position: Position{
						X: 0,
						Y: 0,
					},
					Data: NodeData{
						Label:        fmt.Sprintf("%s-nodegroup", clusterName),
						Provider:     "aws",
						ResourceType: "aws_eks_node_group",
						Module:       id,
						IsContainer:  false,
						DriftStatus:  "IN_SYNC",
						Attributes: map[string]interface{}{
							"cluster_name": clusterName,
						},
					},
					ParentID: targetParent,
				}

				// Connect node group -> EKS cluster
				edgeID := fmt.Sprintf("e-%s-%s", ngID, clusterID)
				edgeMap[edgeID] = Edge{
					ID:       edgeID,
					Source:   ngID,
					Target:   clusterID,
					Type:     "smoothstep",
					Animated: true,
				}
			}

			// Connect private subnets to EKS cluster
			subnetsToConnect := createdPrivateSubnets
			if len(subnetsToConnect) == 0 {
				subnetsToConnect = createdSubnets
			}
			for _, subID := range subnetsToConnect {
				edgeID := fmt.Sprintf("e-%s-%s", subID, clusterID)
				edgeMap[edgeID] = Edge{
					ID:       edgeID,
					Source:   subID,
					Target:   clusterID,
					Type:     "smoothstep",
					Animated: true,
				}
			}

			n.Data.IsContainer = true
			nodeMap[id] = n
		}
	}

	return nodeMap, edgeMap
}

func extractStringList(val interface{}) []string {
	if val == nil {
		return nil
	}
	var res []string
	switch v := val.(type) {
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				res = append(res, s)
			}
		}
	case []string:
		return v
	case string:
		// e.g. ["10.0.1.0/24", "10.0.2.0/24"] or comma-separated
		trimmed := strings.Trim(v, "[] \t\n\r")
		if trimmed != "" {
			parts := strings.Split(trimmed, ",")
			for _, p := range parts {
				cleanP := strings.Trim(p, "\"' \t\n\r")
				if cleanP != "" {
					res = append(res, cleanP)
				}
			}
		}
	}
	return res
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
