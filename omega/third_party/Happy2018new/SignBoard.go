package Happy2018new

import (
	"encoding/json"
	"fmt"
	"math"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/omega/defines"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
)

type SignBoard struct {
	*defines.BasicComponent
	TagName        string   `json:"标签名称"`
	StartPos       [2]int32 `json:"起始坐标"`
	EndPos         [2]int32 `json:"终止坐标"`
	ConstPos       float64  `json:"不变坐标"`
	ConstIsX       bool     `json:"不变轴是X"`
	NumMax         int      `json:"每列最多允许数量"`
	TriggerMessage string   `json:"触发词"`
	PermissionList []string `json:"授权列"`
	msgGet         string
	length         int32
	width          int32
	initPoint      [2]float32
	PermissionMap  map[string]bool
}

func (o *SignBoard) Init(settings *defines.ComponentConfig, storage defines.StorageAndLogProvider) {
	marshal, _ := json.Marshal(settings.Configs)
	if err := json.Unmarshal(marshal, o); err != nil {
		panic(err)
	}
	for key, value := range o.StartPos {
		if value > o.EndPos[key] {
			o.StartPos[key] = o.EndPos[key]
			o.EndPos[key] = value
		}
	}
	pterm.Info.Printf("Init: DEBUG - o.StartPos = %#v, o.EndPos = %#v\n", o.StartPos, o.EndPos)
	o.length = o.EndPos[0] - o.StartPos[0]
	o.width = o.EndPos[1] - o.StartPos[1]
	if o.length < o.width {
		tmp := o.length
		o.length = o.width
		o.width = tmp
	}
	o.length++
	o.width++
	pterm.Info.Printf("Init: DEBUG - length = %#v, width = %#v\n", o.length, o.width)
	if o.StartPos[0] < 0 {
		o.initPoint = [2]float32{float32(o.StartPos[0]) - 0.5, float32(o.StartPos[1])}
	} else {
		o.initPoint = [2]float32{float32(o.StartPos[0]), float32(o.StartPos[1])}
	}
	pterm.Info.Printf("Init: DEBUG - o.initPoint = %#v\n", o.initPoint)
	o.PermissionMap = map[string]bool{}
	for _, value := range o.PermissionList {
		o.PermissionMap[value] = true
	}
	pterm.Info.Printf("Init: DEBUG - o.PermissionMap = %#v\n", o.PermissionMap)
}

func (o *SignBoard) MainFunc() {
	if strings.Contains(o.msgGet, fmt.Sprintf("%v::requestNewSignBoard", o.TriggerMessage)) {
		o.Frame.GetGameControl().SendCmdAndInvokeOnResponse(fmt.Sprintf("testfor @e[tag=\"%v\"]", o.TagName), func(output *packet.CommandOutput) {
			if output.SuccessCount <= 0 {
				pterm.Error.Printf("MainFunc: target entity not found; packet.CommandOutput = %#v\n", output)
				return
			}
			entityList := strings.Split(output.OutputMessages[0].Parameters[0], ", ")
			// get entity list
			lineCount := int(math.Ceil(float64(len(entityList)) / float64(o.NumMax)))
			widthDistance := float32(float32(o.width) / float32(int32(lineCount)+1))
			currentLineCount := 0
			currentPos := [2]float32{o.initPoint[0], o.initPoint[1]}
			currentEntity := -1
			// prepare
			for i := 0; i < lineCount; i++ {
				currentPos[1] = currentPos[1] + widthDistance
				// prepare
				if lineCount > 1 && i < lineCount-1 {
					currentLineCount = o.NumMax
				} else if lineCount == 1 {
					currentLineCount = len(entityList)
				} else {
					currentLineCount = len(entityList) - i*o.NumMax
				}
				// get current line count
				tmp := currentPos[0]
				for j := 0; j < currentLineCount; j++ {
					currentPos[0] = currentPos[0] + float32(o.length)/(float32(currentLineCount)+1)
					currentEntity++
					if o.ConstIsX {
						posx := strconv.FormatFloat(o.ConstPos, 'f', -1, 32)
						posy := strconv.FormatFloat(float64(currentPos[1]), 'f', -1, 32)
						posz := strconv.FormatFloat(float64(currentPos[0]), 'f', -1, 32)
						if !strings.Contains(posx, ".") {
							posx = posx + ".0"
						}
						if !strings.Contains(posy, ".") {
							posy = posy + ".0"
						}
						if !strings.Contains(posz, ".") {
							posz = posz + ".0"
						}
						command := fmt.Sprintf("tp @e[name=\"%v\",c=1,tag=\"%v\"] %v %v %v", entityList[currentEntity], o.TagName, posx, posy, posz)
						pterm.Info.Printf("MainFunc: DEBUG - command = %#v\n", command)
						o.Frame.GetGameControl().SendCmd(command)
					} else {
						posx := strconv.FormatFloat(float64(currentPos[0]), 'f', -1, 32)
						posy := strconv.FormatFloat(float64(currentPos[1]), 'f', -1, 32)
						posz := strconv.FormatFloat(o.ConstPos, 'f', -1, 32)
						if !strings.Contains(posx, ".") {
							posx = posx + ".0"
						}
						if !strings.Contains(posy, ".") {
							posy = posy + ".0"
						}
						if !strings.Contains(posz, ".") {
							posz = posz + ".0"
						}
						command := fmt.Sprintf("tp @e[name=\"%v\",c=1,tag=\"%v\"] %v %v %v", entityList[currentEntity], o.TagName, posx, posy, posz)
						pterm.Info.Printf("MainFunc: DEBUG - command = %#v\n", command)
						o.Frame.GetGameControl().SendCmd(command)
					}
				}
				currentPos[0] = tmp
			}
		})
	} else if strings.Contains(o.msgGet, fmt.Sprintf("%v::getNowSignBoardNameList", o.TriggerMessage)) {
		o.Frame.GetGameControl().SendCmdAndInvokeOnResponse(fmt.Sprintf("testfor @e[tag=\"%v\"]", o.TagName), func(output *packet.CommandOutput) {
			if output.SuccessCount <= 0 {
				pterm.Error.Printf("MainFunc: target entity not found; packet.CommandOutput = %#v\n", output)
				return
			}
			pterm.Info.Printf("packet.CommandOutput.OutputMessages[0].Parameters[0] = %#v\n", strings.Split(output.OutputMessages[0].Parameters[0], ", "))
			// get entity list
		})
	}
}

func (o *SignBoard) Inject(frame defines.MainFrame) {
	o.Frame = frame
	o.Frame.GetGameListener().SetOnTypedPacketCallBack(packet.IDText, func(p packet.Packet) {
		pk := p.(*packet.Text)
		_, ok := o.PermissionMap[pk.SourceName]
		if pk.TextType == 1 && ok {
			o.msgGet = pk.Message
			o.MainFunc()
		}
	})
}
