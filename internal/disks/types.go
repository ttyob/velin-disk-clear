package disks

type Volume struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	MountPoint string `json:"mount_point"`
	FileSystem string `json:"file_system"`
	TotalBytes uint64 `json:"total_bytes"`
	FreeBytes  uint64 `json:"free_bytes"`
	UsedBytes  uint64 `json:"used_bytes"`
	System     bool   `json:"system"`
	Ready      bool   `json:"ready"`
}

func List() ([]Volume, error) {
	return list()
}
