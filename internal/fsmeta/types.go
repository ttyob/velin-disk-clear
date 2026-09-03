package fsmeta

import "os"

type Metadata struct {
	LogicalSize   int64  `json:"logical_size"`
	AllocatedSize int64  `json:"allocated_size"`
	VolumeID      string `json:"volume_id"`
	FileID        string `json:"file_id"`
	LinkCount     uint32 `json:"link_count"`
}

func Read(path string, info os.FileInfo) Metadata {
	return read(path, info)
}
