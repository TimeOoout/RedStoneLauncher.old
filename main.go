package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/winterssy/sreq"
	"os"
	"runtime"
	"strconv"
)

// assetsindex_files

//var header = req.Header{
//	"Accept-Encoding": "gzip",
//	"Connection":      "keep-alive",
//	"Accept-Language": "zh-CN,zh;q=0.8,en-US;q=0.5,en;q=0.3",
//	"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.5112.102 Safari/537.36 Edg/104.0.1293.70",
//}

func main() {

	cpuNum := runtime.NumCPU() //获得当前设备的cpu核心数
	fmt.Println("cpu核心数:", cpuNum)
	runtime.GOMAXPROCS(cpuNum)

	//Default返回一个默认的路由引擎

	os.Remove(".\\RSL.json")
	init_setting()

	Update_global_variables()
	Upadate_Game_List()

	r := gin.Default()
	//验证代理是否存在
	r.POST("/is_exist", func(c *gin.Context) {
		fmt.Printf("[SYSTEM]: " + "INFO: Verify presence." + "\n")
		c.JSON(200, gin.H{"INFO": 0})
	})
	//获取所有版本信息
	r.POST("/get_all_versions", func(c *gin.Context) {
		fmt.Printf("[SYSTEM]: " + "INFO: Get all the MineCraft versions." + "\n")
		data := make(map[string]interface{})
		sreq.Get("https://launchermeta.mojang.com/mc/game/version_manifest.json").JSON(&data)
		c.JSON(200, data)
	})
	//获取LiteLoader列表
	r.POST("/get_LiteLoader_list", func(c *gin.Context) {
		data := make(map[string]interface{})
		sreq.Get("https://bmclapi2.bangbang93.com/liteloader/list?mcversion=" + c.PostForm("Version")).JSON(&data)
		c.JSON(200, data)
	})
	//获取forge列表
	r.POST("/get_forge_list", func(c *gin.Context) {
		data, _ := sreq.Get("https://bmclapi2.bangbang93.com/forge/minecraft/" + c.PostForm("Version")).Text()
		c.JSON(200, data)
	})
	//获取forge支持的MineCraft版本
	r.POST("/get_forge_supported_list", func(c *gin.Context) {
		data, _ := sreq.Get("https://bmclapi2.bangbang93.com/forge/minecraft").Text()
		c.JSON(200, data)
	})
	//获取Optifine列表
	r.POST("/get_optifine_list", func(c *gin.Context) {

		data, _ := sreq.Get("https://bmclapi2.bangbang93.com/optifine/" + c.PostForm("Version")).Text()
		c.JSON(200, data)
	})
	//获取Java列表
	r.POST("/get_java_list", func(c *gin.Context) {

		data, _ := sreq.Get("https://bmclapi2.bangbang93.com/java/list").Text()
		c.JSON(200, data)
	})
	//下载Minecraft
	r.POST("/download", func(c *gin.Context) {
		retry_times, _ := strconv.Atoi(c.PostForm("RetryTimes"))
		path := c.PostForm("WorkPath")
		threads := c.PostForm("Threads")
		threads_ := 4
		if threads != "" {
			threads_, _ = strconv.Atoi(threads)
		}
		res := Download_game(c.PostForm("MCVersion"), c.PostForm("VersionName"), retry_times, DownLoadSource, path, threads_)
		c.String(200, res)
	})

	//启动MineCraft
	r.POST("/execute", func(c *gin.Context) {
		if Current_User == "" {
			c.String(200, "Please select a user!")
		} else {
			java_path := c.PostForm("JavaPath")
			version := c.PostForm("MCVersion")
			version_name := c.PostForm("VersionName")
			user_name := c.PostForm("UserName")
			path := c.PostForm("WorkPath")
			if !Execute_game(version, version_name, user_name, java_path, DownLoadSource, DownloadRetryTimes, path, true) {
				c.String(200, "Unable to execute Minecraft!")
			}
		}
	})
	//获取Java列表
	r.POST("/java_list", func(c *gin.Context) {
		javalist, err := findSystemJava()
		if err != nil {
			fmt.Print(err.Error())
			c.String(502, err.Error())
		}
		c.String(200, javalist)
	})
	//添加离线用户
	r.POST("/adduser_offline", func(c *gin.Context) {
		username := c.PostForm("UserName")
		uuid := c.PostForm("Uuid")
		if uuid == "" {
			uuid_hash := sha256.New()
			byte_username := []byte(username)
			uuid_hash.Write(byte_username)
			uuid_bytes := uuid_hash.Sum(nil)
			uuid = hex.EncodeToString(uuid_bytes)
		}
		if gjson.Get(RSLSettings, "Users.#.#(Name=="+username+")#").Exists() {
			c.String(200, "This user already exists!")
		} else {
			num := gjson.Get(RSLSettings, "Users.#").String()
			if num == "" {
				num = "0"
			}
			set_settings("Users."+num+".Name", username)
			set_settings("Users."+num+".Uuid", uuid)
			set_settings("Users."+num+".Type", "OFFLINE")
			set_settings("Selected", username)
			c.String(200, "")
		}

	})
	//切换当前用户
	r.POST("/Select_user", func(c *gin.Context) {
		username := c.PostForm("UserName")
		if gjson.Get(RSLSettings, "Users.#(Name="+username+")#.Name").String() == "[\""+username+"\"]" {
			set_settings("Selected", username)
			Current_User = gjson.Get(gjson.Get(RSLSettings, "Users.#(Name="+username+")#").String(), "0").String()
			c.String(200, "")
		} else {
			c.String(200, "This user doesn't exists!")
		}
	})
	//获取用户列表
	r.POST("/get_userlist", func(c *gin.Context) {
		c.String(200, gjson.Get(RSLSettings, "Users").String())
	})
	//切换下载源
	r.POST("/change_source", func(c *gin.Context) {
		source := c.PostForm("Source")
		if source == "OFFICIAL" {
			set_settings("DownloadSource", "OFFICIAL")
			DownLoadSource = 0
			c.String(200, "Current source: OFFICIAL")
		} else if source == "BMCL" {
			set_settings("DownloadSource", "BMCL")
			DownLoadSource = 1
			c.String(200, "Current source: BMCL")
		} else {
			set_settings("DownloadSource", "MCBBS")
			DownLoadSource = 2
			c.String(200, "Current source: MCBBS")
		}
	})
	//查看当前可用Java
	r.POST("/web_java_list", func(c *gin.Context) {
		c.String(200, gjson.Get(RSLSettings, "JavaList").String())
	})
	//删除用户
	r.POST("/delete_user", func(c *gin.Context) {
		del_settings("Users.#(Name=" + c.PostForm("UserName") + ")#")
		c.String(200, "")
	})
	//获取已安装列表
	r.POST("/update_game_list", func(c *gin.Context) {
		c.String(200, Game_Dirs)
	})
	//添加游戏路径
	r.POST("/add_game_dir", func(c *gin.Context) {
		data, _ := c.GetRawData()
		body := string(data)
		err := add_game_dir(gjson.Get(body, "Path").String(), gjson.Get(body, "Name").String())
		if err != nil {
			c.String(200, "Unable to add game dir! Error:"+err.Error())
		} else {
			c.String(200, "")
		}
	})
	/*
		运行代理
	*/
	r.Run(":30713")
}
