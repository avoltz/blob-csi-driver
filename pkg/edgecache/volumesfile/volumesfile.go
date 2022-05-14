package volumesfile

import (
	"bufio"
	"os"
)

type VolumeMap map[string]bool

type VolumesFile struct {
	filepath string
	volumes  VolumeMap
}

func NewVolumesFile(filepath string) *VolumesFile {
	f := VolumesFile{
		filepath: filepath,
		volumes:  VolumeMap{},
	}
	return &f
}

func (f *VolumesFile) Check() error {
	_, err := os.Stat(f.filepath)
	if os.IsNotExist(err) {
		return f.Save()
	} else {
		return f.Load()
	}
}

func (f *VolumesFile) Load() error {
	file, err := os.Open(f.filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	volumes := VolumeMap{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		volumes[scanner.Text()] = true
	}
	f.volumes = volumes
	return nil
}

func (f *VolumesFile) Save() error {
	file, err := os.Create(f.filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for volume_id, _ := range f.volumes {
		_, err = writer.WriteString(volume_id + "\n")
		if err != nil {
			return err
		}
	}
	return err
}

func (f *VolumesFile) Add(volume_id string) {
	f.volumes[volume_id] = true
}

func (f *VolumesFile) Remove(volume_id string) bool {
	_, exists := f.volumes[volume_id]
	if exists {
		delete(f.volumes, volume_id)
	}
	return exists
}
