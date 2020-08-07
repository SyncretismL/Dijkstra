package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"sort"
)

//Структура входных данных из файла users.json
type user struct {
	Nick        string       `json:"Nick"`
	Email       string       `json:"Email"`
	Created     string       `json:"Created_at"`
	Subscribers []subscriber `json:"Subscribers"`
}

type subscriber struct {
	Email   string `json:"Email"`
	Created string `json:"Created_at"`
}

//Структура результата работы
type result struct {
	ID   int          `json:"id"`
	From string       `json:"from"`
	To   string       `json:"to"`
	Path []singlePath `json:"path,omitempty"`
}
type singlePath struct {
	Email   string `json:"email"`
	Created string `json:"created_at"`
}

//Зададим структуры для ребер и вершин

type graph struct {
	Edges []*edge
	Nodes []*node
}

type edge struct {
	Parent *node
	Child  *node
	Cost   int
}

type node struct {
	Name    string
	Created string
}

type nodeInfo struct {
	Cost int
	path [][]string
}

const infinity = math.MaxInt64

func main() {
	var g graph

	userWay, err := readCsv("input.csv")
	if err != nil {
		log.Fatal(err)
	}

	users, err := readJSON("users.json")
	if err != nil {
		log.Fatal(err)
	}

	mapNodePointers := g.addGraph(users)

	result := g.formResult(userWay, mapNodePointers)

	name := "result1.json"

	err = createResult(result, name)
	if err != nil {
		log.Fatal(err)
	}
}

//Формирование результата
func (g *graph) formResult(userWay map[int]map[string]string, mapNodePointers map[string]*node) []result {
	result := make([]result, len(userWay))

	var id int

	for i := 0; i < len(userWay); i++ {
		way := userWay[i]
		shortestPath := g.dijkstra(mapNodePointers[way["to"]], mapNodePointers[way["from"]])
		shortestPathSorted := reverseSlice(shortestPath)
		id++
		result[i].ID = id
		result[i].From = way["from"]
		result[i].To = way["to"]

		for _, j := range shortestPathSorted {
			result[i].Path = append(result[i].Path, singlePath{
				Email:   j[0],
				Created: j[1],
			})
		}
	}

	return result
}

// Добавление ребра в граф
func (g *graph) addEdge(parent, child *node, cost int) {
	edge := &edge{
		Parent: parent,
		Child:  child,
		Cost:   cost,
	}

	g.Edges = append(g.Edges, edge)
	g.addNode(parent)
	g.addNode(child)
}

// Добавление вершины, в роли вершины выступает Email
func (g *graph) addNode(node *node) {
	var isPresent bool

	for _, n := range g.Nodes {
		if n == node {
			isPresent = true
		}
	}

	if !isPresent {
		g.Nodes = append(g.Nodes, node)
	}
}

// Заполнение структуры графа (вершины и ребра)
func (g *graph) addGraph(users []user) map[string]*node {
	mapNodePointers := make(map[string]*node, len(users))

	for _, user := range users {
		node := &node{Name: user.Email, Created: user.Created}
		mapNodePointers[user.Email] = node
	}

	for _, user := range users {
		for _, subscr := range user.Subscribers {
			g.addEdge(mapNodePointers[user.Email], mapNodePointers[subscr.Email], 1)
		}
	}

	return mapNodePointers
}

//формирование матрицы, где 'цена' стартовой точки равна 0, а 'цена' перехода в другие - Inf
func (g *graph) newCostTable(startNode *node) map[*node]*nodeInfo {
	costTable := make(map[*node]*nodeInfo)

	costTable[startNode] = &nodeInfo{
		Cost: 0,
		path: nil,
	}

	for _, node := range g.Nodes {
		if node != startNode {
			costTable[node] = &nodeInfo{
				Cost: infinity,
				path: nil,
			}
		}
	}

	return costTable
}

//Получение ребер вершины
func (g *graph) getNodeEdges(node *node) (edges []*edge) {
	for _, edge := range g.Edges {
		if edge.Parent == node {
			edges = append(edges, edge)
		}
	}

	return edges
}

//Получаем ближайшую по 'цене' вершину
func getClosestNonVisitedNode(costTable map[*node]*nodeInfo, visited []*node) *node {
	type CostTableToSort struct {
		Node *node
		Cost int
	}

	var sorted []CostTableToSort

	// Делаем проверку была ли вершина уже посещена
	for node, cost := range costTable {
		var isVisited bool

		for _, visitedNode := range visited {
			if node == visitedNode {
				isVisited = true
			}
		}

		// Если не посещена, то добавляем
		if !isVisited {
			sorted = append(sorted, CostTableToSort{node, cost.Cost})
		}
	}

	//Получаем вершину с минимальной 'ценой' из таблицы путем сортировки
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Cost < sorted[j].Cost
	})

	return sorted[0].Node
}

func (g *graph) dijkstra(startNode *node, endNode *node) (shortestPath [][]string) {
	costTable := g.newCostTable(startNode)

	var visited []*node

	// Запускаем цикл для прохода по всем вершинам
	for len(visited) != len(g.Nodes) {
		// Берем самую ближайшую вершину из матрицы
		node := getClosestNonVisitedNode(costTable, visited)

		// Помечаем вершину как посещенную
		visited = append(visited, node)

		// Запрашиваем ребра для текущей вершины
		nodeEdges := g.getNodeEdges(node)

		for _, edge := range nodeEdges {
			//Подсчет "цены" пути до следующей вершины
			distanceToNeighbor := costTable[node].Cost + edge.Cost

			distanceToNeighbor = positiveCheck(distanceToNeighbor)

			if distanceToNeighbor < costTable[edge.Child].Cost {
				// Обновляем длину в матрице
				costTable[edge.Child].Cost = distanceToNeighbor
				if node != startNode {
					costTable[edge.Child].path = costTable[node].path
					costTable[edge.Child].path = append(costTable[edge.Child].path, []string{node.Name, node.Created})
				}
			}
		}
	}

	// Возвращаем только тот путь, который соответсвует стартовой и конечной вершине
	for node, nodeInfo := range costTable {
		if node.Name == endNode.Name {
			shortestPath = nodeInfo.path
		}
	}

	return shortestPath
}

//Чтение файлов .csv
func readFile(name string) (*os.File, error) {
	File, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("can't find csv: %s", err)
	}

	return File, nil
}

func readJSON(name string) ([]user, error) {
	jsonFile, err := readFile(name)
	if err != nil {
		return nil, err
	}

	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("can't read json: %s", err)
	}

	var users []user

	err = json.Unmarshal(byteValue, &users)
	if err != nil {
		return nil, fmt.Errorf("can't unmarshal json: %s", err)
	}

	if users == nil {
		return nil, fmt.Errorf("no users in file")
	}

	return users, nil
}

func readCsv(name string) (map[int]map[string]string, error) {
	userWay := make(map[int]map[string]string)

	var i int

	input, err := readFile(name)
	if err != nil {
		return nil, fmt.Errorf("can't read csv: %s", err)
	}

	defer input.Close()

	inputReader := csv.NewReader(input)

	for {
		line, err := inputReader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("can't read user line: %s", err)
		}

		userWay[i] = map[string]string{
			"from": line[0],
			"to":   line[1],
		}
		i++
	}

	if userWay == nil {
		return nil, fmt.Errorf("no user ways")
	}

	return userWay, nil
}

func createResult(result []result, name string) error {
	file, err := json.MarshalIndent(result, "  ", "    ")
	if err != nil {
		return fmt.Errorf("marshal failed: %s", err)
	}

	err = ioutil.WriteFile(name, file, 0644)
	if err != nil {
		return fmt.Errorf("write file failed: %s", err)
	}

	return nil
}

func positiveCheck(n int) int {
	if n < 0 {
		return infinity
	}

	return n
}

func reverseSlice(n [][]string) [][]string {
	for i, j := 0, len(n)-1; i < j; i, j = i+1, j-1 {
		n[i], n[j] = n[j], n[i]
	}

	return n
}
