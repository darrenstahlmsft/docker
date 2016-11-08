package opts

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	mounttypes "github.com/docker/docker/api/types/mount"
	units "github.com/docker/go-units"
)

// MountOpt is a Value type for parsing mounts
type MountOpt struct {
	values []mounttypes.Mount
}

// Set a new mount value
func (m *MountOpt) Set(value string) error {
	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return err
	}

	mount := mounttypes.Mount{}

	volumeOptions := func() *mounttypes.VolumeOptions {
		if mount.VolumeOptions == nil {
			mount.VolumeOptions = &mounttypes.VolumeOptions{
				Labels: make(map[string]string),
			}
		}
		if mount.VolumeOptions.DriverConfig == nil {
			mount.VolumeOptions.DriverConfig = &mounttypes.Driver{}
		}
		return mount.VolumeOptions
	}

	bindOptions := func() *mounttypes.BindOptions {
		if mount.BindOptions == nil {
			mount.BindOptions = new(mounttypes.BindOptions)
		}
		return mount.BindOptions
	}

	setValueOnMap := func(target map[string]string, value string) {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) == 1 {
			target[value] = ""
		} else {
			target[parts[0]] = parts[1]
		}
	}

	mount.Type = mounttypes.TypeVolume // default to volume mounts
	// Set writable as the default
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		key := strings.ToLower(parts[0])

		if len(parts) == 1 {
			switch key {
			case "readonly", "ro":
				mount.ReadOnly = true
				continue
			case "volume-nocopy":
				volumeOptions().NoCopy = true
				continue
			}
		}

		if len(parts) != 2 {
			return fmt.Errorf("invalid field '%s' must be a key=value pair", field)
		}

		value := parts[1]
		switch key {
		case "type":
			mount.Type = mounttypes.Type(strings.ToLower(value))
		case "source", "src":
			mount.Source = value
		case "target", "dst", "destination":
			mount.Target = value
		case "readonly", "ro":
			mount.ReadOnly, err = strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %s", key, value)
			}
		case "max-bandwidth":
			var bandwidth int64
			bandwidth, err = units.RAMInBytes(value)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %s", key, value)
			}
			mount.MaxBandwidth = uint64(bandwidth)
		case "max-iops":
			maxiops, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %s", key, value)
			}
			mount.MaxIOps = uint64(maxiops)
		case "bind-propagation":
			bindOptions().Propagation = mounttypes.Propagation(strings.ToLower(value))
		case "volume-nocopy":
			volumeOptions().NoCopy, err = strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("invalid value for populate: %s", value)
			}
		case "volume-label":
			setValueOnMap(volumeOptions().Labels, value)
		case "volume-driver":
			volumeOptions().DriverConfig.Name = value
		case "volume-opt":
			if volumeOptions().DriverConfig.Options == nil {
				volumeOptions().DriverConfig.Options = make(map[string]string)
			}
			setValueOnMap(volumeOptions().DriverConfig.Options, value)
		default:
			return fmt.Errorf("unexpected key '%s' in '%s'", key, field)
		}
	}

	if mount.Type == "" {
		return fmt.Errorf("type is required")
	}

	if mount.Target == "" {
		return fmt.Errorf("target is required")
	}

	if mount.Type == mounttypes.TypeBind && mount.VolumeOptions != nil {
		return fmt.Errorf("cannot mix 'volume-*' options with mount type '%s'", mounttypes.TypeBind)
	}
	if mount.Type == mounttypes.TypeVolume && mount.BindOptions != nil {
		return fmt.Errorf("cannot mix 'bind-*' options with mount type '%s'", mounttypes.TypeVolume)
	}

	m.values = append(m.values, mount)
	return nil
}

// Type returns the type of this option
func (m *MountOpt) Type() string {
	return "mount"
}

// String returns a string repr of this option
func (m *MountOpt) String() string {
	mounts := []string{}
	for _, mount := range m.values {
		repr := fmt.Sprintf("%s %s %s", mount.Type, mount.Source, mount.Target)
		mounts = append(mounts, repr)
	}
	return strings.Join(mounts, ", ")
}

// Value returns the mounts
func (m *MountOpt) Value() []mounttypes.Mount {
	return m.values
}
