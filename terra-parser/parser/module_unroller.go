package parser

import (
	"fmt"
	"strings"
)

// UnrollModularArchitecture unrolls high-level AWS modules (VPC, EKS, S3, DynamoDB)
// and their local wrappers into concrete AWS architectural topology with VPC containers,
// public/private subnets, EKS clusters & node groups, and standalone cloud services.
func UnrollModularArchitecture(nodeMap map[string]Node, edgeMap map[string]Edge, tfVars map[string]interface{}) (map[string]Node, map[string]Edge) {
	if nodeMap == nil {
		return nodeMap, edgeMap
	}

	if tfVars == nil {
		tfVars = make(map[string]interface{})
	}

	// 1. Resolve all node attributes with tfvars
	for id, n := range nodeMap {
		if n.Data.Attributes != nil {
			n.Data.Attributes = ResolveAttributesWithVars(n.Data.Attributes, tfVars)
			nodeMap[id] = n
		}
	}

	modulesToRemove := make([]string, 0)
	newNodes := make(map[string]Node)
	newEdges := make(map[string]Edge)
	publicSubnetIDs := make([]string, 0)
	privateSubnetIDs := make([]string, 0)

	var vpcContainerID string = "vpc-main"
	var foundVPCModule bool

	// ==========================================
	// Pass 1: Identify and Unroll VPC Module
	// ==========================================
	for id, n := range nodeMap {
		if n.Data.ResourceType != "module" {
			continue
		}

		attrs := n.Data.Attributes
		if attrs == nil {
			attrs = make(map[string]interface{})
		}

		src, _ := attrs["source"].(string)
		isVPC := strings.Contains(src, "vpc") || strings.Contains(id, "vpc") || strings.Contains(strings.ToLower(n.Data.Label), "vpc")

		// If this module already has parsed local child resources, keep it as container
		hasLocalChildResources := false
		for _, child := range nodeMap {
			if child.ParentID == id {
				hasLocalChildResources = true
				break
			}
		}
		if hasLocalChildResources {
			continue
		}

		if isVPC {
			foundVPCModule = true
			modulesToRemove = append(modulesToRemove, id)

			vpcName := "prod-vpc"
			if val := getAttrOrVar(attrs, tfVars, "name", "vpc_name"); val != "" {
				vpcName = val
			}

			cidr := "10.0.0.0/16"
			if val := getAttrOrVar(attrs, tfVars, "cidr", "vpc_cidr", "cidr_block"); val != "" {
				cidr = val
			}

			// 1. Create Parent Container VPC Node
			vpcLabel := fmt.Sprintf("VPC (%s | %s)", vpcName, cidr)
			newNodes[vpcContainerID] = Node{
				ID:   vpcContainerID,
				Type: "infrastructureNode",
				Position: Position{
					X: 0,
					Y: 0,
				},
				Data: NodeData{
					Label:        vpcLabel,
					Provider:     "aws",
					ResourceType: "aws_vpc",
					IsContainer:  true,
					DriftStatus:  "IN_SYNC",
					Attributes: map[string]interface{}{
						"cidr_block": cidr,
						"name":       vpcName,
					},
				},
				ParentID: "", // Root container
			}

			// 2. Extract AZs and Create AZ Container Nodes inside VPC
			azs := getListAttrOrVar(attrs, tfVars, "azs")
			if len(azs) == 0 {
				azs = []string{"us-east-1a", "us-east-1b"}
			}

			azNodeIDs := make([]string, len(azs))
			for i, azName := range azs {
				azID := fmt.Sprintf("az-%s", azName)
				azNodeIDs[i] = azID
				newNodes[azID] = Node{
					ID:   azID,
					Type: "infrastructureNode",
					Position: Position{
						X: 0,
						Y: 0,
					},
					Data: NodeData{
						Label:        fmt.Sprintf("AZ %s", azName),
						Provider:     "aws",
						ResourceType: "aws_availability_zone",
						Module:       vpcContainerID,
						IsContainer:  true,
						DriftStatus:  "IN_SYNC",
						Attributes: map[string]interface{}{
							"availability_zone": azName,
							"vpc_id":            vpcContainerID,
						},
					},
					ParentID: vpcContainerID,
				}
			}

			// 3. Extract Public Subnets (assigned to respective AZ container)
			publicSubnets := getListAttrOrVar(attrs, tfVars, "public_subnets")
			if len(publicSubnets) == 0 {
				publicSubnets = []string{"10.0.1.0/24", "10.0.2.0/24"}
			}
			publicSubnetIDs = make([]string, 0, len(publicSubnets))

			for i, subCIDR := range publicSubnets {
				subID := fmt.Sprintf("subnet-pub-%d", i+1)
				publicSubnetIDs = append(publicSubnetIDs, subID)
				targetAZ := azNodeIDs[i%len(azNodeIDs)]
				newNodes[subID] = Node{
					ID:   subID,
					Type: "infrastructureNode",
					Position: Position{
						X: 0,
						Y: 0,
					},
					Data: NodeData{
						Label:        fmt.Sprintf("Public Subnet - %s", subCIDR),
						Provider:     "aws",
						ResourceType: "aws_subnet",
						Module:       vpcContainerID,
						IsContainer:  false,
						DriftStatus:  "IN_SYNC",
						Attributes: map[string]interface{}{
							"cidr_block":        subCIDR,
							"subnet_type":       "public",
							"tier":              "public",
							"vpc_id":            vpcContainerID,
							"availability_zone": azs[i%len(azs)],
						},
					},
					ParentID: targetAZ, // Inside respective AZ container
				}
			}

			// 4. Extract Private Subnets (assigned to respective AZ container)
			privateSubnets := getListAttrOrVar(attrs, tfVars, "private_subnets")
			if len(privateSubnets) == 0 {
				privateSubnets = []string{"10.0.3.0/24", "10.0.4.0/24"}
			}

			for i, subCIDR := range privateSubnets {
				subID := fmt.Sprintf("subnet-priv-%d", i+1)
				privateSubnetIDs = append(privateSubnetIDs, subID)
				targetAZ := azNodeIDs[i%len(azNodeIDs)]
				newNodes[subID] = Node{
					ID:   subID,
					Type: "infrastructureNode",
					Position: Position{
						X: 0,
						Y: 0,
					},
					Data: NodeData{
						Label:        fmt.Sprintf("Private Subnet - %s", subCIDR),
						Provider:     "aws",
						ResourceType: "aws_subnet",
						Module:       vpcContainerID,
						IsContainer:  false,
						DriftStatus:  "IN_SYNC",
						Attributes: map[string]interface{}{
							"cidr_block":        subCIDR,
							"subnet_type":       "private",
							"tier":              "private",
							"vpc_id":            vpcContainerID,
							"availability_zone": azs[i%len(azs)],
						},
					},
					ParentID: targetAZ, // Inside respective AZ container
				}
			}

			// 5. Synthesize Internet Gateway attached to VPC and connect to Public Subnets
			igwID := "igw-main"
			newNodes[igwID] = Node{
				ID:   igwID,
				Type: "infrastructureNode",
				Position: Position{
					X: 0,
					Y: 0,
				},
				Data: NodeData{
					Label:        fmt.Sprintf("%s-igw", vpcName),
					Provider:     "aws",
					ResourceType: "aws_internet_gateway",
					Module:       vpcContainerID,
					IsContainer:  false,
					DriftStatus:  "IN_SYNC",
					Attributes: map[string]interface{}{
						"vpc_id": vpcContainerID,
					},
				},
				ParentID: vpcContainerID,
			}

			// Connect IGW -> Public Subnets
			for _, pubID := range publicSubnetIDs {
				edgeID := fmt.Sprintf("e-%s-%s", igwID, pubID)
				newEdges[edgeID] = Edge{
					ID:       edgeID,
					Source:   igwID,
					Target:   pubID,
					Type:     "smoothstep",
					Animated: true,
				}
			}
			break
		}
	}

	// ==========================================
	// Pass 2: Identify and Unroll EKS Module
	// ==========================================
	for id, n := range nodeMap {
		if n.Data.ResourceType != "module" {
			continue
		}

		attrs := n.Data.Attributes
		if attrs == nil {
			attrs = make(map[string]interface{})
		}

		src, _ := attrs["source"].(string)
		isEKS := strings.Contains(src, "eks") || strings.Contains(id, "eks") || strings.Contains(strings.ToLower(n.Data.Label), "eks")

		// If this module already has parsed local child resources, keep it as container
		hasLocalChildResources := false
		for _, child := range nodeMap {
			if child.ParentID == id {
				hasLocalChildResources = true
				break
			}
		}
		if hasLocalChildResources {
			continue
		}

		if isEKS {
			modulesToRemove = append(modulesToRemove, id)

			eksName := "production-eks"
			if val := getAttrOrVar(attrs, tfVars, "cluster_name", "eks_name", "name"); val != "" {
				eksName = val
			}

			clusterID := "aws_eks_cluster"
			ngID := "aws_eks_node_group"

			targetParent := ""
			if foundVPCModule {
				targetParent = vpcContainerID
			}

			newNodes[clusterID] = Node{
				ID:   clusterID,
				Type: "infrastructureNode",
				Position: Position{
					X: 0,
					Y: 0,
				},
				Data: NodeData{
					Label:        fmt.Sprintf("EKS: %s", eksName),
					Provider:     "aws",
					ResourceType: "aws_eks_cluster",
					Module:       vpcContainerID,
					IsContainer:  false,
					DriftStatus:  "IN_SYNC",
					Attributes: map[string]interface{}{
						"cluster_name": eksName,
						"vpc_id":       targetParent,
					},
				},
				ParentID: targetParent,
			}

			newNodes[ngID] = Node{
				ID:   ngID,
				Type: "infrastructureNode",
				Position: Position{
					X: 0,
					Y: 0,
				},
				Data: NodeData{
					Label:        fmt.Sprintf("Node Group: %s", eksName),
					Provider:     "aws",
					ResourceType: "aws_eks_node_group",
					Module:       vpcContainerID,
					IsContainer:  false,
					DriftStatus:  "IN_SYNC",
					Attributes: map[string]interface{}{
						"cluster_name": eksName,
						"vpc_id":       targetParent,
					},
				},
				ParentID: targetParent,
			}

			// Edge: EKS Cluster -> Node Group
			edgeID := fmt.Sprintf("e-%s-%s", clusterID, ngID)
			newEdges[edgeID] = Edge{
				ID:       edgeID,
				Source:   clusterID,
				Target:   ngID,
				Type:     "smoothstep",
				Animated: true,
			}

			// Edges: Public Subnets -> EKS Cluster (Control Plane)
			for _, pubID := range publicSubnetIDs {
				subEdgeID := fmt.Sprintf("e-%s-%s", pubID, clusterID)
				newEdges[subEdgeID] = Edge{
					ID:       subEdgeID,
					Source:   pubID,
					Target:   clusterID,
					Type:     "smoothstep",
					Animated: true,
				}
			}

			// Edges: EKS Cluster -> Private Subnets (Worker nodes / data tier)
			for _, privID := range privateSubnetIDs {
				privEdgeID := fmt.Sprintf("e-%s-%s", clusterID, privID)
				newEdges[privEdgeID] = Edge{
					ID:       privEdgeID,
					Source:   clusterID,
					Target:   privID,
					Type:     "smoothstep",
					Animated: true,
				}
			}
		}
	}

	// ==========================================
	// Pass 3: Identify and Unroll S3 / DynamoDB Modules
	// ==========================================
	for id, n := range nodeMap {
		if n.Data.ResourceType != "module" {
			continue
		}

		attrs := n.Data.Attributes
		if attrs == nil {
			attrs = make(map[string]interface{})
		}

		src, _ := attrs["source"].(string)
		isS3 := strings.Contains(src, "s3") || strings.Contains(id, "s3")
		isDB := strings.Contains(src, "dynamodb") || strings.Contains(id, "dynamodb")

		if isS3 {
			bucketName := "data-lake-bucket"
			if val := getAttrOrVar(attrs, tfVars, "bucket", "bucket_name", "name"); val != "" {
				bucketName = val
			}

			s3Node := n
			s3Node.Data.Label = bucketName
			s3Node.Data.Provider = "aws"
			s3Node.Data.ResourceType = "aws_s3_bucket"
			s3Node.Data.IsContainer = false
			s3Node.Data.Attributes["bucket"] = bucketName
			s3Node.ParentID = "" // Standalone outside VPC

			nodeMap[id] = s3Node
		} else if isDB {
			tableName := "app-state-table"
			if val := getAttrOrVar(attrs, tfVars, "name", "table_name"); val != "" {
				tableName = val
			}

			dbNode := n
			dbNode.Data.Label = tableName
			dbNode.Data.Provider = "aws"
			dbNode.Data.ResourceType = "aws_dynamodb_table"
			dbNode.Data.IsContainer = false
			dbNode.Data.Attributes["name"] = tableName
			dbNode.ParentID = "" // Standalone outside VPC

			nodeMap[id] = dbNode
		}
	}

	// Remove unrolled module cards
	for _, modID := range modulesToRemove {
		delete(nodeMap, modID)
	}

	// Add all newly created unrolled nodes and edges
	for k, v := range newNodes {
		nodeMap[k] = v
	}
	for k, v := range newEdges {
		edgeMap[k] = v
	}

	return nodeMap, edgeMap
}

func getAttrOrVar(attrs map[string]interface{}, vars map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if val, exists := attrs[k]; exists && val != nil {
			resolved := ResolveVarValue(val, vars)
			if strVal, ok := resolved.(string); ok && strVal != "" {
				return strVal
			}
		}
		if val, exists := vars[k]; exists && val != nil {
			if strVal, ok := val.(string); ok && strVal != "" {
				return strVal
			}
		}
	}
	return ""
}

func getListAttrOrVar(attrs map[string]interface{}, vars map[string]interface{}, keys ...string) []string {
	for _, k := range keys {
		if val, exists := attrs[k]; exists && val != nil {
			resolved := ResolveVarValue(val, vars)
			list := extractStringList(resolved)
			if len(list) > 0 {
				return list
			}
		}
		if val, exists := vars[k]; exists && val != nil {
			list := extractStringList(val)
			if len(list) > 0 {
				return list
			}
		}
	}
	return nil
}
