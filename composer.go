package main

import (
	"encoding/json"
	"github.com/jinzhu/copier"
)

func optimizeComposerVersions(versions []map[string]interface{}) []map[string]interface{} {
	newVersionList := make([]map[string]interface{}, 0)

	var firstVersion map[string]interface{}

	for _, version := range versions {
		if len(firstVersion) == 0 {
			copier.Copy(&firstVersion, &version)

			newVersionList = append(newVersionList, version)
			continue
		}

		newVersion := make(map[string]interface{})

		for key, val := range version {
			firstVersionVal, ok := firstVersion[key]

			if !ok {
				firstVersion[key] = val
				newVersion[key] = val
			} else {
				firstVal, _ := json.Marshal(firstVersionVal)
				secondVal, _ := json.Marshal(val)

				if string(firstVal) != string(secondVal) {
					firstVersion[key] = val
					newVersion[key] = val
				}
			}
		}

		for key, _ := range firstVersion {
			_, ok := version[key]

			if !ok {
				newVersion[key] = "__unset"
				delete(firstVersion, key)
			}
		}

		newVersionList = append(newVersionList, newVersion)
	}

	return newVersionList
}
