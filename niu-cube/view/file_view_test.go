package view

import (
	"fmt"
	"testing"
	"time"
)

type FileViewer struct {
	components []*Component //有哪些组件
	Width      int
	Height     int
	Layout
	FilePath string
	compMap  map[string]*Component
}

func (v *FileViewer) getComponentById(id string) *Component {
	if value, ok := v.compMap[id]; ok {
		return value
	}
	return nil
}

func (v *FileViewer) joinRoom(token string) {
	//start....
}

type Component struct {
	ID string
	EventListener
}

//定义component的事件
type OnClick func()

type OnFileChanged func(originFile, newFile string)

type OnDrop func(x, y, z int64)

type EventListener struct {
	OnClick
	OnFileChanged
	OnDrop
}

//todo 这一步要抽象出来
func (c *Component) initAndroidViewEventListener() {
	// findById(c.ID).addListener()
	c.EventListener = EventListener{
		OnClick: func() {
			fmt.Println("default click....")
		},
	}
	go func() {
		timer := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-timer.C:
				c.EventListener.OnClick()
			}
		}
	}()
}

//布局设置
type Layout struct {
}

type FileViewerConfig struct {
	Width    int
	Height   int
	RollPage bool //是否支持滚动翻页
	Layout
}

//
func InitFileViewer(config *FileViewerConfig, filePath string) *FileViewer {
	viewer := &FileViewer{
		components: nil,
		Width:      config.Width,
		Height:     config.Height,
		FilePath:   filePath,
		Layout:     Layout{},
		compMap:    make(map[string]*Component),
	}
	viewer.components = initComponentsByConfig(config)
	viewer.compMap[viewer.components[0].ID] = viewer.components[0]
	return viewer
}

//根据配置初始化对应的组件
func initComponentsByConfig(config *FileViewerConfig) []*Component {
	result := make([]*Component, 1)
	if config.RollPage { //增加滚动翻页组件
		rollC := &Component{
			ID: "roll_page_component",
		}
		rollC.initAndroidViewEventListener() //这里绑定各个平台的事件
		result[0] = rollC
	}
	return result
}

//文件浏览器。。。
func TestFileViewer(t *testing.T) {
	//设置参数
	config := &FileViewerConfig{
		Width:    600,
		Height:   500,
		RollPage: true,
	}
	//初始化文件浏览器
	viewer := InitFileViewer(config, "path://")
	//注册事件
	viewer.getComponentById("roll_page_component").EventListener.OnClick = func() {
		fmt.Println("roll_page_component click....")
	}
	viewer.getComponentById("roll_page_component").EventListener.OnDrop = func(x, y, z int64) {

	}
	viewer.joinRoom("")

	time.Sleep(5 * time.Second)
}
