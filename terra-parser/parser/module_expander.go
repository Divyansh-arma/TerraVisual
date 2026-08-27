package parser

import (
	"fmt"
	"strings"
)

// DecomposeSemanticModules transforms high-level Terraform AWS module calls
// (VPC, EKS, S3, DynamoDB) into visual infrastructure topology with VPC containers,
// public/private subnets, EKS clusters & node groups, and standalone cloud services.
func DecomposeSemanticModules(nodeMap map[string]Node, edgeMap map[string]Edge) (map[string]Node, map[string]Edge) {
	if nodeMap == nil {
		return nodeMap, edgeMap
	}

	// 1. First Pass: Find or synthesize VPC containers
	var defaultVPCID string
	for id, n := range nodeMap {
		if n.Data.ResourceType == "aws_vpc" {
			defaultVPCID = id
			break
		}
	}

	// Store module IDs to remove if they were decomposed into child resources
	modulesToRemove := make([]string, 0)
	newNodes := make(map[string]Node)
	newEdges := make(map[string]Edge)

	for id, n := range nodeMap {
		if n.Data.ResourceType != "module" {
			continue
		}

		attrs := n.Data.Attributes
		if attrs == nil {
			attrs = make(map[string]interface{})
		}

		src, _ := attrs["source"].(string)
		modName := strings.TrimPrefix(id, "module.")

		// Skip local directory modules (e.g. ./modules/vpc, ./modules/eks) which are parsed directly from local .tf files
		if strings.HasPrefix(src, ".") || strings.HasPrefix(src, "/") {
			continue
		}

		// ==========================================
		// 1. terraform-aws-modules/vpc/aws
		// ==========================================
		if strings.Contains(src, "aws-modules/vpc") || strings.Contains(src, "terraform-aws-modules/vpc") || (strings.Contains(id, "vpc") && attrs["cidr"] != nil) {
			vpcName := "AWS VPC"
			if nameVal, ok := attrs["name"].(string); ok && nameVal != "" {
				vpcName = nameVal
			} else if nameVal, ok := attrs["vpc_name"].(string); ok && nameVal != "" {
				vpcName = nameVal
			}

			cidr := "10.0.0.0/16"
			if c, ok := attrs["cidr"].(string); ok && c != "" {
				cidr = c
			} else if c, ok := attrs["cidr_block"].(string); ok && c != "" {
				cidr = c
			}

			// Transform this module into the primary VPC container node
			vpcNode := n
			vpcNode.Data.Label = fmt.Sprintf("%s (%s)", vpcName, cidr)
			vpcNode.Data.Provider = "aws"
			vpcNode.Data.ResourceType = "aws_vpc"
			vpcNode.Data.IsContainer = true
			vpcNode.Data.Attributes["cidr_block"] = cidr
			vpcNode.Data.Attributes["name"] = vpcName
			vpcNode.ParentID = "" // Top-level container

			nodeMap[id] = vpcNode
			defaultVPCID = id
			vpcNodeID := id

			// Extract AZs
			azs := extractStringList(attrs["azs"])
			if len(azs) == 0 {
				azs = []string{"us-east-1a", "us-east-1b", "us-east-1c"}
			}

			// Generate Public Subnets
			publicSubnets := extractStringList(attrs["public_subnets"])
			for i, subCIDR := range publicSubnets {
				subID := fmt.Sprintf("%s.aws_subnet.public_%d", vpcNodeID, i+1)
				az := azs[i%len(azs)]
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
						Module:       vpcNodeID,
						IsContainer:  false,
						DriftStatus:  "IN_SYNC",
						Attributes: map[string]interface{}{
							"cidr_block":        subCIDR,
							"subnet_type":       "public",
							"availability_zone": az,
							"vpc_id":            vpcNodeID,
						},
					},
					ParentID: vpcNodeID,
				}
			}

			// Generate Private Subnets
			privateSubnets := extractStringList(attrs["private_subnets"])
			for i, subCIDR := range privateSubnets {
				subID := fmt.Sprintf("%s.aws_subnet.private_%d", vpcNodeID, i+1)
				az := azs[i%len(azs)]
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
						Module:       vpcNodeID,
						IsContainer:  false,
						DriftStatus:  "IN_SYNC",
						Attributes: map[string]interface{}{
							"cidr_block":        subCIDR,
							"subnet_type":       "private",
							"availability_zone": az,
							"vpc_id":            vpcNodeID,
						},
					},
					ParentID: vpcNodeID,
				}
			}

			// Synthesize Internet Gateway attached to VPC
			igwID := fmt.Sprintf("%s.aws_internet_gateway.this", vpcNodeID)
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
					Module:       vpcNodeID,
					IsContainer:  false,
					DriftStatus:  "IN_SYNC",
					Attributes: map[string]interface{}{
						"vpc_id": vpcNodeID,
					},
				},
				ParentID: vpcNodeID,
			}
			continue
		}

		// ==========================================
		// 2. terraform-aws-modules/eks/aws
		// ==========================================
		if strings.Contains(src, "aws-modules/eks") || strings.Contains(src, "terraform-aws-modules/eks") || (strings.Contains(id, "eks") && attrs["cluster_name"] != nil) {
			clusterName := "EKS Cluster"
			if cn, ok := attrs["cluster_name"].(string); ok && cn != "" {
				clusterName = cn
			} else if cn, ok := attrs["name"].(string); ok && cn != "" {
				clusterName = cn
			}

			targetVPC := defaultVPCID

			clusterID := fmt.Sprintf("%s.aws_eks_cluster.this", id)
			ngID := fmt.Sprintf("%s.aws_eks_node_group.this", id)

			newNodes[clusterID] = Node{
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
						"vpc_id":       targetVPC,
					},
				},
				ParentID: targetVPC, // Nested inside the VPC container
			}

			newNodes[ngID] = Node{
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
						"vpc_id":       targetVPC,
					},
				},
				ParentID: targetVPC,
			}

			// Edge: EKS Cluster -> EKS Node Group
			edgeID := fmt.Sprintf("e-%s-%s", clusterID, ngID)
			newEdges[edgeID] = Edge{
				ID:       edgeID,
				Source:   clusterID,
				Target:   ngID,
				Type:     "smoothstep",
				Animated: true,
			}

			// Connect Private Subnets from VPC -> EKS Cluster
			for subID, subNode := range newNodes {
				if subNode.ParentID == targetVPC && subNode.Data.ResourceType == "aws_subnet" && subNode.Data.Attributes["subnet_type"] == "private" {
					subEdgeID := fmt.Sprintf("e-%s-%s", subID, clusterID)
					newEdges[subEdgeID] = Edge{
						ID:       subEdgeID,
						Source:   subID,
						Target:   clusterID,
						Type:     "smoothstep",
						Animated: true,
					}
				}
			}
			for subID, subNode := range nodeMap {
				if subNode.ParentID == targetVPC && subNode.Data.ResourceType == "aws_subnet" && subNode.Data.Attributes["subnet_type"] == "private" {
					subEdgeID := fmt.Sprintf("e-%s-%s", subID, clusterID)
					newEdges[subEdgeID] = Edge{
						ID:       subEdgeID,
						Source:   subID,
						Target:   clusterID,
						Type:     "smoothstep",
						Animated: true,
					}
				}
			}

			modulesToRemove = append(modulesToRemove, id)
			continue
		}

		// ==========================================
		// 3. terraform-aws-modules/s3-bucket/aws
		// ==========================================
		if strings.Contains(src, "s3-bucket") || strings.Contains(src, "aws-modules/s3") || (strings.Contains(id, "s3") && attrs["bucket"] != nil) {
			bucketName := modName
			if b, ok := attrs["bucket"].(string); ok && b != "" {
				bucketName = b
			}

			s3Node := n
			s3Node.Data.Label = bucketName
			s3Node.Data.Provider = "aws"
			s3Node.Data.ResourceType = "aws_s3_bucket"
			s3Node.Data.IsContainer = false
			s3Node.ParentID = "" // Outside VPC on global canvas

			nodeMap[id] = s3Node
			continue
		}

		// ==========================================
		// 4. terraform-aws-modules/dynamodb-table/aws
		// ==========================================
		if strings.Contains(src, "dynamodb-table") || strings.Contains(src, "aws-modules/dynamodb") || (strings.Contains(id, "dynamodb") && attrs["name"] != nil) {
			tableName := modName
			if t, ok := attrs["name"].(string); ok && t != "" {
				tableName = t
			}

			dbNode := n
			dbNode.Data.Label = tableName
			dbNode.Data.Provider = "aws"
			dbNode.Data.ResourceType = "aws_dynamodb_table"
			dbNode.Data.IsContainer = false
			dbNode.ParentID = "" // Outside VPC on global canvas

			nodeMap[id] = dbNode
			continue
		}
	}

	// Remove decomposed module wrappers
	for _, modID := range modulesToRemove {
		delete(nodeMap, modID)
	}

	// Merge newly created nodes and edges
	for k, v := range newNodes {
		nodeMap[k] = v
	}
	for k, v := range newEdges {
		edgeMap[k] = v
	}

	return nodeMap, edgeMap
}
