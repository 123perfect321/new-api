package model

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// EnsureOAuthGroup makes a department group usable by OIDC users.
// Existing channel groups are extended only when they already contain default.
func EnsureOAuthGroup(groupName string) error {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" || groupName == "default" {
		return nil
	}

	ratios := ratio_setting.GetGroupRatioCopy()
	if _, ok := ratios[groupName]; !ok {
		ratios[groupName] = 1
		encoded, err := json.Marshal(ratios)
		if err != nil {
			return err
		}
		if err := UpdateOption("group_ratio_setting.group_ratio", string(encoded)); err != nil {
			return err
		}
		if err := ratio_setting.UpdateGroupRatioByJSONString(string(encoded)); err != nil {
			return err
		}
	}

	var channels []Channel
	if err := DB.Find(&channels).Error; err != nil {
		return err
	}
	for i := range channels {
		groups := channels[i].GetGroups()
		foundGroup := false
		foundDefault := false
		for _, group := range groups {
			if group == groupName {
				foundGroup = true
			}
			if group == "default" {
				foundDefault = true
			}
		}
		if foundGroup || !foundDefault {
			continue
		}
		groups = append(groups, groupName)
		channels[i].Group = strings.Join(groups, ",")
		if err := channels[i].Update(); err != nil {
			return err
		}
	}
	return nil
}
