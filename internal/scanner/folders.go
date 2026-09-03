package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strings"
)

type folderAccumulator struct {
	node     *Folder
	children map[string]*folderAccumulator
}

func buildFolders(items []Item) []*Folder {
	roots := make(map[string]*folderAccumulator)
	for _, item := range items {
		volume := filepath.VolumeName(item.Path)
		if volume == "" {
			volume = string(filepath.Separator)
		}
		root := roots[volume]
		if root == nil {
			root = newFolder(volume, volume)
			roots[volume] = root
		}
		addToFolder(root, item)
	}
	result := make([]*Folder, 0, len(roots))
	for _, root := range roots {
		finaliseFolder(root)
		result = append(result, root.node)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func newFolder(name, path string) *folderAccumulator {
	hash := sha256.Sum256([]byte(path))
	return &folderAccumulator{
		node:     &Folder{ID: hex.EncodeToString(hash[:10]), Name: name, Path: path, HighestRisk: "low"},
		children: make(map[string]*folderAccumulator),
	}
}

func addToFolder(root *folderAccumulator, item Item) {
	root.node.FileCount++
	root.node.LogicalBytes += item.LogicalSize
	root.node.AllocatedBytes += item.AllocatedSize
	root.node.EstimatedReclaimable += item.EstimatedReclaimable
	root.node.ItemIDs = append(root.node.ItemIDs, item.ID)
	root.node.HighestRisk = maxRisk(root.node.HighestRisk, item.Risk)

	directory := filepath.Clean(item.Directory)
	volume := filepath.VolumeName(directory)
	relative := strings.TrimPrefix(directory, volume)
	relative = strings.Trim(relative, string(filepath.Separator))
	if relative == "" || relative == "." {
		return
	}
	current := root
	currentPath := volume + string(filepath.Separator)
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		currentPath = filepath.Join(currentPath, part)
		child := current.children[part]
		if child == nil {
			child = newFolder(part, currentPath)
			current.children[part] = child
		}
		child.node.FileCount++
		child.node.LogicalBytes += item.LogicalSize
		child.node.AllocatedBytes += item.AllocatedSize
		child.node.EstimatedReclaimable += item.EstimatedReclaimable
		child.node.ItemIDs = append(child.node.ItemIDs, item.ID)
		child.node.HighestRisk = maxRisk(child.node.HighestRisk, item.Risk)
		current = child
	}
}

func finaliseFolder(accumulator *folderAccumulator) {
	accumulator.node.Children = make([]*Folder, 0, len(accumulator.children))
	for _, child := range accumulator.children {
		finaliseFolder(child)
		accumulator.node.Children = append(accumulator.node.Children, child.node)
	}
	sort.Slice(accumulator.node.Children, func(i, j int) bool {
		return accumulator.node.Children[i].AllocatedBytes > accumulator.node.Children[j].AllocatedBytes
	})
}

func maxRisk(left, right string) string {
	if riskRank(right) > riskRank(left) {
		return right
	}
	return left
}
