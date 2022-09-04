package main

import (
	"errors"
	"fmt"
	"github.com/tidwall/sjson"
	"io/ioutil"
	"os"
)

func init_setting() bool {
	_, err := os.Stat("RSL.json")
	if !os.IsNotExist(err) {
		if get_settings() == false {
			return false
		}
	} else {
		if get_settings() == false {
			return false
		}
		javalist, er := findSystemJava()
		if er != nil {
			fmt.Print("\n", "Unable to get java list :", err.Error(), "\n")
		} else {
			set_settings("JavaList.0", javalist)
			set_settings("GameDir.0.WorkPath.Dir", Current_path+"\\.minecraft")
			set_settings("SelectPath", Current_path+"\\.minecraft")
			set_settings("About.Version", Launcher_Version)
			set_settings("DownloadSource", "MCBBS")
			set_settings("RetryTimes", "5")
			if system_info == "windows" {
				homepath, hper := os.UserHomeDir()
				if hper == nil {
					set_settings("OfficialGameDir", homepath+"\\AppData\\Roaming\\.minecraft")
				}
			}

		}
	}

	return true
}

func get_settings() bool {
	_, err := os.Stat("RSL.json")
	if !os.IsNotExist(err) {
		settins, er := ioutil.ReadFile("RSL.json")
		if er != nil {
			fmt.Print("\n", "Unable to open RSL.json!", "\n")
			return false
		}
		RSLSettings = string(settins)
	} else {
		settingfile, e := os.OpenFile("RSL.json", os.O_CREATE|os.O_TRUNC, 0666)
		defer settingfile.Close()
		if e != nil {
			fmt.Print("\n", "Failed to create RSL.json!", "\n")
			return false
		}
	}
	return true
}

func set_settings(path string, value string) {
	RSLSettings, _ = sjson.Set(RSLSettings, path, value)
	write_settings(RSLSettings)
}
func del_settings(path string) {
	RSLSettings, _ = sjson.Delete(RSLSettings, path)
	write_settings(RSLSettings)
}

func add_game_dir(path string, name string) error {
	_, er := os.Stat(path)
	if os.IsNotExist(er) {
		return errors.New("This path does not exist!")
	}
	if name == "" {
		return errors.New("Invalid name!")
	}
	set_settings("GameDir.-1."+name+".Dir", path)
	Upadate_Game_List()
	return nil
}

func write_settings(settings string) bool {
	settingfile, e := os.OpenFile("RSL.json", os.O_CREATE|os.O_TRUNC, 0666)
	defer settingfile.Close()
	if e != nil {
		fmt.Print("\n", "Failed to open RSL.json!", "\n")
		return false
	}
	_, er := settingfile.WriteString(settings)
	if er != nil {
		fmt.Print("\n", "Failed to write to RSL.json!", "\n")
		return false
	}
	return false
}
