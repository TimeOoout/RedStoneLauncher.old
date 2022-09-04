package main

import (
	"github.com/tidwall/gjson"
	"strconv"
)

func Update_global_variables() {
	//获取下载源
	if gjson.Get(RSLSettings, "DownloadSource").String() == "MCBBS" {
		DownLoadSource = 2
	} else if gjson.Get(RSLSettings, "DownloadSource").String() == "BMCL" {
		DownLoadSource = 1
	} else {
		DownLoadSource = 0
	}
	//获取重试次数
	ret, ret_err := strconv.Atoi(gjson.Get(RSLSettings, "RetryTimes").String())
	if ret_err == nil {
		if ret > 0 {
			DownloadRetryTimes = ret
		}
	}
	//更新游戏列表
	Upadate_Game_List()

}

func Upadate_Game_List() {
	game_num := int(gjson.Get(RSLSettings, "GameDir.#").Int())
	for i := 0; i < game_num; i++ {
		game_dir_name := gjson.Get(RSLSettings, "GameDir."+strconv.Itoa(i)).String()
		Info(game_dir_name)
		//if err != nil {
		//	Error("Unable to get game list '" + gjson.Get(RSLSettings, "GameDir."+strconv.Itoa(i)).String() + "',Error:" + err.Error())
		//	continue
		//}

	}
}
