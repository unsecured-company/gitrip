package utils

import (
	"strings"
)

type TreeNode struct {
	Name     string
	Children map[string]*TreeNode
}

func newTreeNode(name string) *TreeNode {
	return &TreeNode{
		Name:     name,
		Children: make(map[string]*TreeNode),
	}
}

func (n *TreeNode) AddPath(path string) {
	parts := strings.Split(path, "/")
	current := n
	for _, part := range parts {
		if _, exists := current.Children[part]; !exists {
			current.Children[part] = newTreeNode(part)
		}
		current = current.Children[part]
	}
}

func GetTreeAsString(paths []string) string {
	tree := buildTree(paths)
	return treeIntoString(tree, "")
}

func buildTree(paths []string) *TreeNode {
	root := newTreeNode("/")

	for _, path := range paths {
		root.AddPath(path)
	}

	return root
}

func treeIntoString(node *TreeNode, prefix string) (str string) {
	str += prefix + node.Name + "\n"

	for _, child := range node.Children {
		str += treeIntoString(child, prefix+"    ")
	}

	return
}
