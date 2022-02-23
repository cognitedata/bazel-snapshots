package bazel

type BuildEventOutput struct {
	ID struct {
		TargetCompleted struct {
			Label string `json:"label"`
		}
	}
	Completed struct {
		Success      bool `json:"success"`
		OutputGroups []struct {
			Name string `json:"name"`
		} `json:"outputGroup"`
		ImportantOutput []ImportantOutput `json:"importantOutput"`
	}
}

type ImportantOutput struct {
	Name       string   `json:"name"`
	URI        string   `json:"uri"`
	PathPrefix []string `json:"pathPrefix"`
}
