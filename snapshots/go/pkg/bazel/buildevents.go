/* Copyright 2022 Cognite AS */

package bazel

type BuildEventOutput struct {
	ID struct {
		NamedSet struct {
			ID string `json:"id"`
		}
		TargetCompleted struct {
			Label string `json:"label"`
		}
	}

	NamedSetOfFiles NamedSetOfFiles `json:"namedSetOfFiles"`

	Completed struct {
		Success      bool `json:"success"`
		OutputGroups []struct {
			Name     string `json:"name"`
			FileSets []struct {
				ID string `json:"id"`
			} `json:"fileSets"`
		} `json:"outputGroup"`
	}
}

type NamedSetOfFiles struct {
	Files    []NamedSetOfFilesFile    `json:"files"`
	FileSets []NamedSetOfFilesFileSet `json:"fileSets"`
}

type NamedSetOfFilesFile struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type NamedSetOfFilesFileSet struct {
	ID string `json:"id"`
}
