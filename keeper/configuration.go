package keeper

import (
	"errors"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

type VideoConfiguration struct {
	Enabled           bool   `toml:"enabled"`
	WorkerName        string `toml:"worker_name"`
	WorkerKeyLocation string `toml:"worker_key_location"`
	MinReward         int64  `toml:"min_reward"`
	GPUAmount         int64  `toml:"gpu_amount"`
	Path              string
}

func GetVideoRenderingConfiguration(rootPath string) (*VideoConfiguration, error) {
	var path string = rootPath + "/config/videoRendering.toml"
	conf := VideoConfiguration{Enabled: false, Path: path}

	// Load the YAML file
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			conf.SaveConf()
			return &conf, nil
		}

		log.Fatalf("Unable to open VideoRendering configuration file. %v", err.Error())
		return nil, err
	}
	defer file.Close()

	decoder := toml.NewDecoder(file)
	if _, err := decoder.Decode(&conf); err != nil {
		log.Fatalf("Failed to decode YAML: %v\n", err.Error())
		return nil, err
	}

	return &conf, nil
}

func (c *VideoConfiguration) SaveConf() error {
	// Marshal the struct into YAML format
	data, err := toml.Marshal(&c)
	if err != nil {
		log.Fatalf("Error marshaling to YAML: %v\n", err)
		return err
	}

	// Save the YAML data to a file
	file, err := os.Create(c.Path)
	if err != nil {
		log.Fatalf("Error creating file: %v\n", err.Error())
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		log.Fatalf("Error writing to file: %v\n", err)
		return err
	}

	log.Println("YAML data saved to " + c.Path)
	return nil
}
