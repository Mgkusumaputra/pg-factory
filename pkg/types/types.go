package types

type Instance struct {
	Container string `json:"container"`
	Volume    string `json:"volume"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Password  string `json:"password"`
	Db        string `json:"db"`
	Version   string `json:"version"`
	CreatedAt string `json:"created_at"`
}

type InstanceList struct {
	Instances []Instance `json:"instances"`
}
