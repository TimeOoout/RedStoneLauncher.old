package main

import (
	"os"
	"runtime"
	"sync"
)

/*


{
    "JavaList":[
        "D:\\JavaJDK\\bin\\java.exe"
    ],
    "GameDir":[
        "D:\\GolangFiles\\RedStoneLauncher\\.minecraft"
    ],
    "About":{
        "Version":"0.0.1_Alpha"
    },
    "DownloadSource":"MCBBS",
    "RetryTimes":"5",
    "OfficialGameDir":"C:\\Users\\Star Dream\\AppData\\Roaming\\.minecraft",
    "Users":[
        {
            "Name":"Tester",
            "Uuid":"9e7cd9cb5a63a3591e16f4d835f32a1c4a84ab66e39ae27aa448c03b66bf63e7",
            "Type":"OFFLINE"
        }
    ],
    "Selected":"Tester"
}



*/

var indexassets_waitgroup sync.WaitGroup
var system_info = runtime.GOOS
var system_arch = runtime.GOARCH

var RSLSettings = "{}"

var Current_path, _ = os.Getwd()
var Launcher_Version = "0.0.1_Alpha"
var Current_User = ""
var Game_Dirs = ""

/*
下载源说明：
<1>:为BMCLAPI
<2>:为MCBBS
*/

var DownLoadSource = 2

var BMCL = "{" +
	"\"VersionList_v1\":\"https://bmclapi2.bangbang93.com/mc/game/version_manifest.json\"," +
	"\"VersionList_v2\":\"https://bmclapi2.bangbang93.com/mc/game/version_manifest_v2.json\"," +
	"\"Index\":\"https://bmclapi2.bangbang93.com/\"," +
	"\"Launcher\":\"https://bmclapi2.bangbang93.com/\"," +
	"\"Assets\":\"https://bmclapi2.bangbang93.com/assets/\"," +
	"\"Libraries\":\"https://bmclapi2.bangbang93.com/maven/\"" +
	"}"

var MCBBS = "{" +
	"\"VersionList_v1\":\"https://download.mcbbs.net/mc/game/version_manifest.json\"," +
	"\"VersionList_v2\":\"https://download.mcbbs.net/mc/game/version_manifest_v2.json\"," +
	"\"Index\":\"https://download.mcbbs.net/\"," +
	"\"Launcher\":\"https://download.mcbbs.net/\"," +
	"\"Assets\":\"https://download.mcbbs.net/assets/\"," +
	"\"Libraries\":\"https://download.mcbbs.net/maven/\"" +
	"}"

var MineCraftSource = "{" +
	"\"VersionList_v1\":\"http://launchermeta.mojang.com/mc/game/version_manifest.json\"," +
	"\"VersionList_v2\":\"http://launchermeta.mojang.com/mc/game/version_manifest_v2.json\"," +
	"\"Index\":\"https://launchermeta.mojang.com/\"," +
	"\"Launcher\":\"https://launcher.mojang.com/\"," +
	"\"Assets\":\"http://resources.download.minecraft.net/\"," +
	"\"Libraries\":\"https://libraries.minecraft.net/\"" +
	"}"

var DownloadRetryTimes = 5

var Current_game_list = ""

var download_limit = NewGoLimit(240)

//NewGoLimit(1280)

var redownload_list = ""
