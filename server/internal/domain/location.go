package domain

import (
	"errors"
	"fmt"
	"strings"
)

// LocationInfo is a parsed VRChat location_id.
type LocationInfo struct {
	WorldID         string
	InstanceID      string
	Region          string
	GroupID         *string
	GroupAccessType *string
	Friends         *string
	Hidden          *string
	Private         *string
}

// ParseLocationID parses a VRChat location_id of the form
// "wrld_xxx:60123~region(jp)~group(grp_yyy)~groupAccessType(members)" and returns
// the structured fields. region is required.
func ParseLocationID(locationID string) (LocationInfo, error) {
	worldPart, rest, _ := strings.Cut(locationID, ":")
	if worldPart == "" || rest == "" {
		return LocationInfo{}, fmt.Errorf("invalid location_id: %q", locationID)
	}

	parts := strings.Split(rest, "~")
	if parts[0] == "" {
		return LocationInfo{}, fmt.Errorf("invalid location_id: %q", locationID)
	}

	info := LocationInfo{
		WorldID:    worldPart,
		InstanceID: parts[0],
	}

	for _, part := range parts[1:] {
		if !strings.HasSuffix(part, ")") {
			continue
		}
		open := strings.Index(part, "(")
		if open < 0 {
			continue
		}
		key := part[:open]
		val := part[open+1 : len(part)-1]
		switch key {
		case "region":
			info.Region = val
		case "group":
			v := val
			info.GroupID = &v
		case "groupAccessType":
			v := val
			info.GroupAccessType = &v
		case "friends":
			v := val
			info.Friends = &v
		case "hidden":
			v := val
			info.Hidden = &v
		case "private":
			v := val
			info.Private = &v
		}
	}

	if info.Region == "" {
		return LocationInfo{}, errors.New("region missing in location_id: " + locationID)
	}

	return info, nil
}
