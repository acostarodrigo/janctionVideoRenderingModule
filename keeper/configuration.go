package keeper

import (
	"errors"
	"io/fs"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

type VideoConfiguration struct {
	Enabled           bool   `toml:"enabled"`
	WorkerName        string `toml:"worker_name"`
	WorkerAddress     string `toml:"worker_address"`
	WorkerKeyLocation string `toml:"worker_key_location"`
	MinReward         int64  `toml:"min_reward"`
	GPUAmount         int64  `toml:"gpu_amount"`
	ConfigPath        string
	RootPath          string
}

func GetVideoRenderingConfiguration(rootPath string) (*VideoConfiguration, error) {
	var configPath string = rootPath + "/config/videoRendering.toml"
	conf := VideoConfiguration{Enabled: false, RootPath: rootPath, ConfigPath: configPath}

	// we make sure the root path exists. It might yet not be initialized
	_, err := os.Stat(rootPath)
	if errors.Is(err, fs.ErrNotExist) {
		return &conf, nil
	}

	// Load the YAML file
	file, err := os.Open(configPath)
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
	// we make sure the root path exists. It might not be initialized
	_, err := os.Stat(c.ConfigPath)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println("File doesn't exists", c.ConfigPath)
		return nil
	}

	// Marshal the struct into YAML format
	data, err := toml.Marshal(&c)
	if err != nil {
		log.Fatalf("Error marshaling to YAML: %v\n", err)
		return err
	}

	// Save the YAML data to a file
	file, err := os.Create(c.ConfigPath)
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

	log.Println("YAML data saved to " + c.ConfigPath)
	return nil
}
