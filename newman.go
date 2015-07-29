package main

import (
	"fmt"
	"github.com/awalterschulze/gographviz"
	"github.com/awalterschulze/gographviz/parser"
	. "github.com/gyuho/goraph/graph"
	"log"
	"math"
)

type Cluster struct {
	Nodes    map[*Node]bool
	InEdges  map[*Node]map[*Node]float64 // クラスタ内エッジ
	OutEdges map[*Node]map[*Node]float64 // クラスタ外エッジ
}

func NewCluster(node *Node) *Cluster {
	cluster := new(Cluster)
	cluster.Nodes = map[*Node]bool{node: true}
	return cluster
}

func visualize(data *Data) {
	graphAst, _ := parser.ParseString(`digraph G {}`)
	g := gographviz.NewGraph()

	gographviz.Analyse(graphAst, g)

	// step1 : add nodes
	for node := range data.NodeMap {
		g.AddNode("G", node.ID, nil)
	}

	// step2 : make edge from source node  to target node
	for _, edge := range data.GetEdges() {
		g.AddEdge(edge.Src.ID, edge.Dst.ID, true, nil)
	}
	output := g.String()
	fmt.Println(output)
}

func countEdgesInCluster(cluster *Cluster) float64 {
	cluster_edges := 0.0
	for _, src := range cluster.InEdges {
		cluster_edges += float64(len(src))
	}
	return cluster_edges
}

func isInCluster(target *Node, cluster *Cluster) bool {
	for node, _ := range cluster.Nodes {
		if target.ID == node.ID {
			return true
		}
	}
	return false
}

func countEdgesBetweenClusters(cluster *Cluster, graph *Data) float64 {
	edge_count := 0.0
	for _, edges := range cluster.OutEdges {
		edge_count += float64(len(edges))
	}
	return edge_count
}

func computeDeltaModularity(c1, c2 *Cluster, graph *Data) float64 {
	all_edges := float64(len(graph.GetEdges()))

	c1_2_edges := 0.0
	for _, c1_edges := range c1.OutEdges {
		for node, value := range c1_edges {
			if isInCluster(node, c2) {
				c1_2_edges += value
			}
		}
	}

	c2_1_edges := 0.0
	for _, c2_edges := range c1.OutEdges {
		for node, value := range c2_edges {
			if isInCluster(node, c1) {
				c2_1_edges += value
			}
		}
	}

	c1_from_edges := countEdgesBetweenClusters(c1, graph)
	c2_from_edges := countEdgesBetweenClusters(c2, graph)

	return ((c1_2_edges / all_edges) + (c2_1_edges / all_edges) - 2*(c1_from_edges/all_edges)*(c2_from_edges/all_edges))
}

func computeModularity(c1, c2 *Cluster, graph *Data) float64 {
	/*
		require:
			- 総エッジ数
			- クラスタ内エッジ数
				- 適宜計算
			- クラスタ外エッジ数
	*/

	// compute number of edges
	all_edges := float64(len(graph.GetEdges()))
	log.Println("all_edges:", all_edges)

	// compute number of edge in clusters
	c1_edges := countEdgesInCluster(c1)
	c2_edges := countEdgesInCluster(c2)
	log.Println("c1_edges:", c1_edges)
	log.Println("c2_edges:", c2_edges)

	// compute nuber of edge between clusters
	c1_from_edges := countEdgesBetweenClusters(c1, graph)
	c2_from_edges := countEdgesBetweenClusters(c2, graph)
	log.Println("c1_from_edges:", c1_from_edges)
	log.Println("c2_from_edges:", c2_from_edges)

	q := ((c1_edges / all_edges) - math.Pow((c1_from_edges/all_edges), 2)) + ((c2_edges / all_edges) - math.Pow((c2_from_edges/all_edges), 2))
	return q
}

func makeCluster(cluster *Cluster, graph *Data) {
	inedges := map[*Node]map[*Node]float64{}
	for n1, _ := range cluster.Nodes {
		inedges[n1] = map[*Node]float64{}
		for n2, _ := range cluster.Nodes {
			if graph.GetEdgeWeight(n1, n2) != 0.0 {
				// クラスタ内エッジ
				inedges[n1][n2] = float64(graph.GetEdgeWeight(n1, n2))
			}
		}
	}
	cluster.InEdges = inedges

	outedges := map[*Node]map[*Node]float64{}
	for n1, _ := range cluster.Nodes {
		outedges[n1] = map[*Node]float64{}
		for node, _ := range graph.NodeMap {
			if !isInCluster(node, cluster) && graph.GetEdgeWeight(n1, node) != 0.0 {
				outedges[n1][node] = float64(graph.GetEdgeWeight(n1, node))
			}
		}
	}
	cluster.OutEdges = outedges
}

func merge(pair []*Cluster, graph *Data) *Cluster {
	cluster := new(Cluster)
	cluster.Nodes = map[*Node]bool{}
	for _, cls := range pair {
		for node, _ := range cls.Nodes {
			cluster.Nodes[node] = true
		}
	}
	makeCluster(cluster, graph)
	return cluster
}

func initClusters(graph *Data) map[*Cluster]bool {
	cluster_map := map[*Cluster]bool{}
	for node, _ := range graph.NodeMap {
		cluster := new(Cluster)
		cluster.Nodes = map[*Node]bool{
			node: true,
		}
		makeCluster(cluster, graph)
		cluster_map[cluster] = true
	}
	return cluster_map
}

func clustering(graph *Data) {
	// step1 : make clusters as node
	cluster_map := initClusters(graph)
	cluster_map_b := initClusters(graph)

	m := 0.0

	// step2 : do newman method
	max_pair := []*Cluster{}
	max_delta := 0.0
	log.Println("before:", len(cluster_map))
	for len(cluster_map) > 1 {
		max_delta := 0.0
		max_pair = []*Cluster{}
		for c1, _ := range cluster_map {
			for c2, _ := range cluster_map {
				delta := computeDeltaModularity(c1, c2, graph)
				if c1 != c2 && delta > max_delta {
					max_pair = []*Cluster{c1, c2}
					max_delta = computeDeltaModularity(c1, c2, graph)
				}
			}
		}
		log.Println("max delta:", max_delta)

		if max_delta >= 0.0 {
			merge_cluster := merge(max_pair, graph)
			delete(cluster_map, max_pair[0])
			delete(cluster_map, max_pair[1])
			cluster_map[merge_cluster] = true
		} else {
			break
		}
		if max_delta > m {
			m = max_delta
			cluster_map_b = map[*Cluster]bool{}
			for k, v := range cluster_map {
				cluster_map_b[k] = v
			}
		}

		log.Println("len:", len(cluster_map))
	}

	fmt.Println("output")
	for cls, _ := range cluster_map {
		for node, _ := range cls.Nodes {
			fmt.Println(node.ID)
		}
		fmt.Println()
	}

	fmt.Println("maximum output")
	for cls, _ := range cluster_map_b {
		for node, _ := range cls.Nodes {
			fmt.Println(node.ID)
		}
		fmt.Println()
	}

	// log.Println("after:", len(cluster_map))
	log.Println("max delta:", max_delta)
}

func makeTestData() *Data {
	data := New()

	// first cluster
	data.Connect(NewNode("A"), NewNode("B"), 1.0)
	data.Connect(NewNode("A"), NewNode("F"), 1.0)
	data.Connect(NewNode("B"), NewNode("A"), 1.0)
	data.Connect(NewNode("B"), NewNode("C"), 1.0)
	data.Connect(NewNode("B"), NewNode("E"), 1.0)
	data.Connect(NewNode("C"), NewNode("B"), 1.0)
	data.Connect(NewNode("C"), NewNode("F"), 1.0)
	data.Connect(NewNode("D"), NewNode("E"), 1.0)
	data.Connect(NewNode("D"), NewNode("F"), 1.0)
	data.Connect(NewNode("E"), NewNode("B"), 1.0)
	data.Connect(NewNode("E"), NewNode("D"), 1.0)
	data.Connect(NewNode("F"), NewNode("A"), 1.0)
	data.Connect(NewNode("F"), NewNode("C"), 1.0)
	data.Connect(NewNode("F"), NewNode("D"), 1.0)

	// second cluster
	data.Connect(NewNode("G"), NewNode("H"), 1.0)
	data.Connect(NewNode("G"), NewNode("K"), 1.0)
	data.Connect(NewNode("H"), NewNode("G"), 1.0)
	data.Connect(NewNode("H"), NewNode("I"), 1.0)
	data.Connect(NewNode("H"), NewNode("L"), 1.0)
	data.Connect(NewNode("H"), NewNode("K"), 1.0)
	data.Connect(NewNode("H"), NewNode("J"), 1.0)
	data.Connect(NewNode("I"), NewNode("H"), 1.0)
	data.Connect(NewNode("I"), NewNode("J"), 1.0)
	data.Connect(NewNode("I"), NewNode("L"), 1.0)
	data.Connect(NewNode("J"), NewNode("H"), 1.0)
	data.Connect(NewNode("J"), NewNode("I"), 1.0)
	data.Connect(NewNode("K"), NewNode("G"), 1.0)
	data.Connect(NewNode("K"), NewNode("H"), 1.0)
	data.Connect(NewNode("L"), NewNode("H"), 1.0)
	data.Connect(NewNode("L"), NewNode("I"), 1.0)

	// edges between clusters
	//data.Connect(NewNode("G"), NewNode("A"), 1.0)
	//data.Connect(NewNode("H"), NewNode("E"), 1.0)
	data.Connect(NewNode("I"), NewNode("F"), 1.0)
	//data.Connect(NewNode("A"), NewNode("G"), 1.0)
	data.Connect(NewNode("E"), NewNode("H"), 1.0)
	//data.Connect(NewNode("F"), NewNode("I"), 1.0)

	// modularity : 0.8194444...
	////c1 := new(Cluster)
	////c1.Nodes = []*Node{
	////	data.GetNodeByID("A"),
	////	data.GetNodeByID("B"),
	////	data.GetNodeByID("C"),
	////	data.GetNodeByID("D"),
	////	data.GetNodeByID("E"),
	////	data.GetNodeByID("F"),
	////}
	////makeCluster(c1, data)

	////c2 := new(Cluster)
	////c2.Nodes = []*Node{
	////	data.GetNodeByID("G"),
	////	data.GetNodeByID("H"),
	////	data.GetNodeByID("I"),
	////	data.GetNodeByID("J"),
	////	data.GetNodeByID("K"),
	////	data.GetNodeByID("L"),
	////}
	////makeCluster(c2, data)
	////return data, c1, c2
	return data
}

func makeFakeData() *Data {
	data := New()
	data.Connect(NewNode("A"), NewNode("B"), 1.0)
	data.Connect(NewNode("A"), NewNode("C"), 1.0)
	data.Connect(NewNode("A"), NewNode("E"), 1.0)
	data.Connect(NewNode("A"), NewNode("F"), 1.0)
	data.Connect(NewNode("C"), NewNode("B"), 1.0)
	data.Connect(NewNode("C"), NewNode("E"), 1.0)
	data.Connect(NewNode("E"), NewNode("F"), 1.0)
	data.Connect(NewNode("F"), NewNode("B"), 1.0)
	data.Connect(NewNode("G"), NewNode("B"), 1.0)
	data.Connect(NewNode("G"), NewNode("H"), 1.0)
	data.Connect(NewNode("H"), NewNode("I"), 1.0)
	data.Connect(NewNode("H"), NewNode("J"), 1.0)
	data.Connect(NewNode("I"), NewNode("G"), 1.0)
	data.Connect(NewNode("J"), NewNode("K"), 1.0)
	data.Connect(NewNode("K"), NewNode("H"), 1.0)
	data.Connect(NewNode("K"), NewNode("I"), 1.0)

	return data
}

func main() {
	// data := makeFakeData()
	// data, c1, c2 := makeTestData()
	data := makeTestData()
	log.Println(data)
	// log.Println("modularity", computeModularity(c1, c2, data))
	clustering(data)

	//	visualize(data)
}
