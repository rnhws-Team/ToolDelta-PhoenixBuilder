package Happy2018new

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	DB "phoenixbuilder/fastbuilder/database"
	GameInterface "phoenixbuilder/game_control/game_interface"
	ResourcesControl "phoenixbuilder/game_control/resources_control"
	"phoenixbuilder/omega/defines"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
)

type RecordPlayerPosition struct {
	*defines.BasicComponent
	apis           GameInterface.GameInterface
	Database       *DB.Database
	LogPassingChan chan SingleLog
	Stoped         chan struct{}
	WaitToStop     sync.WaitGroup
	DatabaseName   string  `json:"数据库名称"`
	CheckTime      int     `json:"检查周期(单位:游戏刻)"`
	OutputToCMD    bool    `json:"是否在控制台实时打印坐标变动记录"`
	RefreshTime    int     `json:"日志最小更新周期(单位:游戏刻)"`
	MinRadius      float32 `json:"单个日志容许的最大活动半径"`
}

// 描述单个日志下单个玩家的坐标信息
type PosInfo struct {
	Dimension byte       // 玩家所处的维度
	Position  [3]float32 // 玩家的位置
	YRot      float32    // 玩家的偏航角
}

// 描述单个日志
type SingleLog struct {
	Time       time.Time // 日志捕获时间
	PlayerName string    // 该玩家的游戏名称
	Pos        PosInfo   // 该玩家的坐标信息
}

// 将 pos_info 编码为二进制形式
func (o *RecordPlayerPosition) Marshal(pos_info PosInfo) (
	result []byte,
	err error,
) {
	buffer := bytes.NewBuffer([]byte{})
	if err = binary.Write(buffer, binary.LittleEndian, pos_info.Dimension); err != nil {
		err = fmt.Errorf("Marshal: %v", err)
		return
	}
	if err = binary.Write(buffer, binary.LittleEndian, pos_info.Position[0]); err != nil {
		err = fmt.Errorf("Marshal: %v", err)
		return
	}
	if err = binary.Write(buffer, binary.LittleEndian, pos_info.Position[1]); err != nil {
		err = fmt.Errorf("Marshal: %v", err)
		return
	}
	if err = binary.Write(buffer, binary.LittleEndian, pos_info.Position[2]); err != nil {
		err = fmt.Errorf("Marshal: %v", err)
		return
	}
	if err = binary.Write(buffer, binary.LittleEndian, pos_info.YRot); err != nil {
		err = fmt.Errorf("Marshal: %v", err)
		return
	}
	return buffer.Bytes(), nil
}

// 将 pos_info 从二进制形式解码
func (o *RecordPlayerPosition) Unmarshal(pos_info []byte) (
	result PosInfo,
	err error,
) {
	reader := bytes.NewBuffer(pos_info)
	if err = binary.Read(reader, binary.LittleEndian, &result.Dimension); err != nil {
		err = fmt.Errorf("Unmarshal: %v", err)
		return
	}
	if err = binary.Read(reader, binary.LittleEndian, &result.Position[0]); err != nil {
		err = fmt.Errorf("Unmarshal: %v", err)
		return
	}
	if err = binary.Read(reader, binary.LittleEndian, &result.Position[1]); err != nil {
		err = fmt.Errorf("Unmarshal: %v", err)
		return
	}
	if err = binary.Read(reader, binary.LittleEndian, &result.Position[2]); err != nil {
		err = fmt.Errorf("Unmarshal: %v", err)
		return
	}
	if err = binary.Read(reader, binary.LittleEndian, &result.YRot); err != nil {
		err = fmt.Errorf("Unmarshal: %v", err)
		return
	}
	return
}

/*
创建并取得对应的(子)存储桶。
已存在时将不会创建。

返回根存储桶 ，
其名为日志所对应日期的时间戳，
用于存储当天内所有玩家产生的日志

返回属于 log.PlayerName 的子存储桶，
其名为 log.PlayerName ，
用于存储玩家 log.PlayerName 所产生的日志

返回用于记录坐标详细变动记录的子存储桶，
其名为 pos_change_details ，
用于存储玩家 log.PlayerName 的坐标详细变动记录。

返回用于储存映射信息的子存储桶，
其名为 pos_change_details_index_mapping ，
这用于为 pos_change_details 建立以时间为标准的映射，
即该存储桶中键为非零自然数且从 1 开始不断递增，
值对应 pos_change_details 中对应的键名，
代表对应的日志以时间顺序排序。

返回用于释放根存储桶的函数，
你必须在使用完各个子存储桶桶调用该函数。

上述根存储桶和子存储桶在使用完毕后必须释放，
否则可能会造成阻塞
*/
func (o *RecordPlayerPosition) CreateAndGetBucket(log *SingleLog) (
	player_info *DB.Bucket,
	pos_change_details *DB.Bucket,
	pos_change_details_index_mapping *DB.Bucket,
	root_bucket_release_func func() error,
	err error,
) {
	var root *DB.Bucket
	// 初始化
	{
		year, months, day := log.Time.Date()
		datetime := time.Time{}
		datetime = datetime.AddDate(year, int(months), day)
		// 截取日志创建的日期
		// 该日期的时间戳将作为根存储桶的名称
		buffer := bytes.NewBuffer([]byte{})
		binary.Write(buffer, binary.LittleEndian, datetime.Unix())
		// 获取时间戳的二进制形式
		if !o.Database.HasBucket(buffer.Bytes()) {
			if err = o.Database.CreateBucket(buffer.Bytes()); err != nil {
				err = fmt.Errorf("CreateAndGetBucket: %v", err)
				return
			}
		}
		// 创建根存储桶，若其未被创建的话
		root = o.Database.GetBucketByName(buffer.Bytes())
		if root == nil {
			err = fmt.Errorf(
				"CreateAndGetBucket: The resulting bucket of root named %s is nil",
				hex.EncodeToString(buffer.Bytes()),
			)
			return
		}
		// 取得根存储桶
		root_bucket_release_func = func() error {
			return root.UseDown()
		}
		// 此函数用于释放根存储桶
	}
	// 创建并取得根存储桶
	{
		if !root.HasKey([]byte(log.PlayerName)) {
			if err = root.CreateSubBucket([]byte(log.PlayerName)); err != nil {
				err = fmt.Errorf("CreateAndGetBucket: %v", err)
				return
			}
		}
		// 创建属于玩家 log.PlayerName 的子存储桶，
		// 若其未被创建的话
		player_info = root.GetSubBucketByName([]byte(log.PlayerName))
		if player_info == nil {
			err = fmt.Errorf(
				"CreateAndGetBucket: The resulting bucket of player_info named %s is nil",
				hex.EncodeToString([]byte(log.PlayerName)),
			)
			return
		}
		// 取得属于玩家 log.PlayerName 的子存储桶
	}
	// 创建并取得属于玩家 log.PlayerName 的子存储桶
	{
		if !player_info.HasKey([]byte("pos_change_details")) {
			if err = player_info.CreateSubBucket([]byte("pos_change_details")); err != nil {
				err = fmt.Errorf("CreateAndGetBucket: %v", err)
				return
			}
		}
		// 创建用于记录坐标详细变动记录的子存储桶，
		// 若其未被创建的话
		pos_change_details = player_info.GetSubBucketByName([]byte("pos_change_details"))
		if pos_change_details == nil {
			err = fmt.Errorf(
				"CreateAndGetBucket: The resulting bucket of pos_change_details named %s is nil",
				hex.EncodeToString([]byte("pos_change_details")),
			)
			return
		}
		// 取得用于记录坐标详细变动记录的子存储桶
	}
	// 创建并取得用于记录坐标详细变动记录的子存储桶
	{
		if !player_info.HasKey([]byte("pos_change_details_index_mapping")) {
			if err = player_info.CreateSubBucket(
				[]byte("pos_change_details_index_mapping"),
			); err != nil {
				err = fmt.Errorf("CreateAndGetBucket: %v", err)
				return
			}
		}
		// 创建用于储存映射信息的子存储桶，
		// 若其未被创建的话
		pos_change_details_index_mapping = player_info.GetSubBucketByName(
			[]byte("pos_change_details_index_mapping"),
		)
		if pos_change_details_index_mapping == nil {
			err = fmt.Errorf(
				"CreateAndGetBucket: The resulting bucket of pos_change_details_index_mapping named %s is nil",
				hex.EncodeToString([]byte("pos_change_details_index_mapping")),
			)
			return
		}
		// 取得用于储存映射信息的子存储桶
	}
	// 创建并取得用于储存映射信息的子存储桶
	return
	// 返回值
}

/*
将 log 记入到数据库。

force_update 指代是否需要强制更新 pos_change_details 字段。
通常情况下，每经过时间 o.RefreshTime 时，
上层函数传入的 force_update 将为真。

特别地，last_info_record 字段将总保持最新
*/
func (o *RecordPlayerPosition) RecordSingleLog(
	log SingleLog,
	force_update bool,
) (
	modified bool,
	err error,
) {
	var sum_counts uint64
	var last_pos_info PosInfo
	// 初始化
	player, details, index, root_bucket_release_func, err := o.CreateAndGetBucket(&log)
	if err != nil {
		err = fmt.Errorf("RecordSingleLog: %v", err)
		return
	}
	defer func() {
		if root_bucket_release_func == nil {
			return
		}
		err := root_bucket_release_func()
		if err != nil {
			panic(fmt.Sprintf("RecordSingleLog: %v", err))
		}
	}()
	// 创建并取得对应的(子)存储桶。
	// 已存在时将不会创建
	pos_info, err := o.Marshal(log.Pos)
	if err != nil {
		err = fmt.Errorf("RecordSingleLog: %v", err)
		return
	}
	log_create_time, err := log.Time.MarshalBinary()
	if err != nil {
		err = fmt.Errorf("RecordSingleLog: %v", err)
		return
	}
	// 将 log.Pos 和 log.Time 编码为二进制形式
	sum_counts_bytes := player.GetDataByKey([]byte("pos_change_details_sum_counts"))
	if sum_counts_bytes == nil {
		player.PutData([]byte("pos_change_details_sum_counts"), []byte{0, 0, 0, 0, 0, 0, 0, 1})
		sum_counts_bytes = []byte{0, 0, 0, 0, 0, 0, 0, 1}
	}
	reader := bytes.NewBuffer(sum_counts_bytes)
	if err = binary.Read(reader, binary.LittleEndian, &sum_counts); err != nil {
		err = fmt.Errorf("RecordSingleLog: %v", err)
		return
	}
	// 获取日志总数
	buffer := bytes.NewBuffer([]byte{})
	if err = binary.Write(buffer, binary.LittleEndian, sum_counts+1); err != nil {
		err = fmt.Errorf("RecordSingleLog: %v", err)
		return
	}
	new_sum_counts_bytes := buffer.Bytes()
	// 获取更新后日志总数所对应的二进制形式。
	// 可能不被使用
	if err = player.PutData(
		[]byte("last_info_record"),
		append(log_create_time, pos_info...),
	); err != nil {
		err = fmt.Errorf("RecordSingleLog: %v", err)
		return
	}
	// 将 last_info_record 更新为 log_create_time 与 pos_info 的合并形式
	{
		key := index.GetDataByKey(sum_counts_bytes)
		if key == nil {
			key = log_create_time
		}
		value := details.GetDataByKey(key)
		// 取得上一次的日志信息，
		// 可能不被使用
		switch {
		case value == nil || force_update:
			if err = details.PutData(log_create_time, pos_info); err != nil {
				err = fmt.Errorf("RecordSingleLog: %v", err)
				return
			}
			modified = true
			// 此时日志将被要求强制更新。
			// 通常发生在创建首个日志，
			// 或已经过 o.RefreshTime 时间时
		case value != nil:
			last_pos_info, err = o.Unmarshal(value)
			if err != nil {
				err = fmt.Errorf("RecordSingleLog: %v", err)
				return
			}
			x, y, z := last_pos_info.Position[0], last_pos_info.Position[1], last_pos_info.Position[2]
			x1, y1, z1 := log.Pos.Position[0], log.Pos.Position[1], log.Pos.Position[2]
			d, d1 := last_pos_info.Dimension, log.Pos.Dimension
			// 加载上一个日志和当前日志的的坐标和维度信息
			if d != d1 || (x-x1)*(x-x1)+(y-y1)*(y-y1)+(z-z1)*(z-z1) > o.MinRadius*o.MinRadius {
				if err = details.PutData(log_create_time, pos_info); err != nil {
					err = fmt.Errorf("RecordSingleLog: %v", err)
					return
				}
				modified = true
			}
			// 将当前维度和上一个维度比对，
			// 同时将当前坐标与上一个坐标比对，
			// 判断是否需要向 pos_change_details 写入新信息
		}
		// 更新 pos_change_details 中的内容
		if modified {
			if err = player.PutData(
				[]byte("pos_change_details_sum_counts"), new_sum_counts_bytes,
			); err != nil {
				err = fmt.Errorf("RecordSingleLog: %v", err)
				return
			}
			if err = index.PutData(new_sum_counts_bytes, log_create_time); err != nil {
				err = fmt.Errorf("RecordSingleLog: %v", err)
				return
			}
		}
		// 更新日志总计数和映射表
	}
	// 将日志信息写入数据库
	return
	// 返回值
}

// 以 o.CheckTime 的周期请求坐标及朝向信息，
// 并在解析和处理后发送至管道 o.LogPassingChan 中。
// 只会在遭遇错误时返回值
func (o *RecordPlayerPosition) ReceiveResponse() error {
	ticker := time.NewTicker(time.Second / 20 * time.Duration(o.CheckTime))
	defer ticker.Stop()
	// 初始化
	for {
		resp := o.apis.SendWSCommandWithResponse(
			"querytarget @a",
			ResourcesControl.CommandRequestOptions{
				TimeOut: time.Second * 5,
			},
		)
		if resp.Error != nil && resp.ErrorType == ResourcesControl.ErrCommandRequestTimeOut {
			<-ticker.C
			continue
		}
		result, err := o.apis.ParseTargetQueryingInfo(resp.Respond)
		if err != nil {
			return fmt.Errorf("ReceiveResponse: %v", err)
		}
		// 请求并解析租赁符返回的玩家坐标及朝向信息
		for _, value := range result {
			player_uuid, err := uuid.Parse(value.UniqueId)
			if err != nil {
				return fmt.Errorf("ReceiveResponse: %v", err)
			}
			temp, _ := strconv.ParseFloat(
				strconv.FormatFloat(float64(value.Position[2]), 'f', 5, 32),
				32,
			)
			value.Position[2] = float32(temp) - 1.62001
			o.LogPassingChan <- SingleLog{
				Time: time.Now(),
				PlayerName: o.Frame.GetGameControl().GetPlayerKitByUUID(
					player_uuid).GetRelatedUQ().Username,
				Pos: PosInfo{
					Dimension: value.Dimension,
					Position:  value.Position,
					YRot:      value.YRot,
				},
			}
		}
		// 向管道发送日志信息
		select {
		case <-ticker.C:
		case <-o.Stoped:
			o.Stoped <- struct{}{}
			return nil
		}
		// 等待下一次检测
	}
}

func (o *RecordPlayerPosition) Init(
	settings *defines.ComponentConfig,
	storage defines.StorageAndLogProvider,
) {
	marshal, _ := json.Marshal(settings.Configs)
	err := json.Unmarshal(marshal, o)
	if err != nil {
		panic(err)
	}
	if o.DatabaseName == "" {
		o.DatabaseName = "PlayerLocation"
	}
	o.LogPassingChan = make(chan SingleLog, 64)
	o.Stoped = make(chan struct{}, 1)
}

func (o *RecordPlayerPosition) Inject(frame defines.MainFrame) {
	var err error
	o.Frame = frame
	o.apis = o.Frame.GetGameControl().GetInteraction()
	o.Database, err = DB.OpenOrCreateDatabase(o.Frame.GetRelativeFileName(o.DatabaseName))
	if err != nil {
		panic(err)
	}
}

func (o *RecordPlayerPosition) Activate() {
	var should_refresh bool
	var modified bool
	var err error
	o.WaitToStop.Add(3)
	go func() {
		for {
			err := o.ReceiveResponse()
			if err == nil {
				o.WaitToStop.Add(-1)
				return
			}
			pterm.Error.Printf("RecordPlayerPosition: %v\n", err)
		}
	}()
	go func() {
		ticker := time.NewTicker(time.Second / 20 * time.Duration(o.RefreshTime))
		defer ticker.Stop()
		for {
			should_refresh = true
			select {
			case <-ticker.C:
			case <-o.Stoped:
				o.Stoped <- struct{}{}
				o.WaitToStop.Add(-1)
				return
			}
		}
	}()
	go func() {
		for {
			var log SingleLog
			select {
			case log = <-o.LogPassingChan:
			case <-o.Stoped:
				o.Stoped <- struct{}{}
				o.WaitToStop.Add(-1)
				return
			}
			if should_refresh {
				modified, err = o.RecordSingleLog(log, true)
				should_refresh = false
			} else {
				modified, err = o.RecordSingleLog(log, false)
			}
			if err != nil {
				pterm.Error.Printf("RecordPlayerPosition: %v\n", err)
			}
			if o.OutputToCMD && modified {
				pterm.Info.Printf("RecordPlayerPosition: %#v\n", log)
			}
		}
	}()
}

func (o *RecordPlayerPosition) Stop() error {
	fmt.Println("正在保存 " + o.DatabaseName)
	o.Stoped <- struct{}{}
	o.WaitToStop.Wait()
	err := o.Database.CloseDatabase()
	if err == nil {
		fmt.Printf("%v 已保存完毕\n", o.DatabaseName)
	}
	return err
}
