package main

import (
	"bufio"
	"flag"
	"fmt"
	// "math" // var start = int64(math.MaxInt64)
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/kuaileniu/zlog"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

func init() {
	zlog.InitLogger(zlog.LogConfig{Filename: "./logs/gotpl.log"})
}

type ProcessInfo struct {
	ProcessName    string // 进程名称
	StartTimeNoStr string // 此进程启动納秒表示的开始时间
	EndTimeNoStr   string // 此进程启动纳秒表示的结束时间
	StartTimeNoInt int64  // 此进程启动納秒表示的开始时间
	EndTimeNoInt   int64  // 此进程启动纳秒表示的结束时间
	StartTime      string // 此进程启动开始时间
	EndTime        string // 此进程启动结束时间
	Duration       string // 耗时
	StartLine      string
	EndLine        string
}

var startTpl = "|XXXXXXXX|ENTER_REGION|XXXXXXXX|starting(XXXXXXXX)|"
var endTpl = "|XXXXXXXX|EXIT_REGION|XXXXXXXX|starting(XXXXXXXX)|"

// 正则匹配 |数字| 形式的时间戳（17~19位，纳秒级）
var re = regexp.MustCompile(`\|(\d{17,19})\|`)

// 北京时间时区 UTC+8
var beijingLoc = time.FixedZone("Beijing", 8*3600)

func main() {
	var logFile, process string
	flag.StringVar(&process, "process", "", "输入进程文件路径 (e.g. process.txt)")
	flag.StringVar(&logFile, "log", "", "输入日志文件路径 (e.g. cupMonitor_converted.log)")
	flag.Parse()

	processNames := load_process(process)
	logLines := load_log(logFile)
	allProcessInfoPtrSli := make([]*ProcessInfo, 0)
	for _, processName := range processNames {
		infoPtrSli := find_one_process(processName, logLines)
		allProcessInfoPtrSli = append(allProcessInfoPtrSli, infoPtrSli...)
	}

	abstract_time(allProcessInfoPtrSli)
	compute_duration(allProcessInfoPtrSli)

	// startTimeBatchSli:=make([]int64,0)
	sortedInfoSli := abstrct_every_startTime(processNames, allProcessInfoPtrSli)
	write_excel(sortedInfoSli)
	zap.L().Info("startTimeBatchSli", zap.Any("sortedInfoSli", sortedInfoSli))
	// zap.L().Info("Compelte:", zap.Any("allProcessInfoPtrSli", allProcessInfoPtrSli))
}

func write_excel(allProcessInfoPtrSli []*ProcessInfo) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	// // 创建一个工作表
	// index, err := f.NewSheet("Sheet2")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// 设置单元格的值
	// f.SetCellValue("Sheet2", "A3", "Hello world.")
	write_one_batch(f, allProcessInfoPtrSli)
	// 设置工作簿的默认工作表
	// f.SetActiveSheet(index - 1)
	// 根据指定路径保存文件
	if err := f.SaveAs("cupMonitor.xlsx"); err != nil {
		fmt.Println(err)
	}
}

func write_one_batch(f *excelize.File, oneBach []*ProcessInfo) {
	sheet := "Sheet1"

	// 创建一个居中对齐的样式
	styleCenter := excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "left",   // 水平居左
			Vertical:   "center", // 垂直居中
		},
	}
	// 将样式添加到工作簿，并获取样式 ID
	styleID, err := f.NewStyle(&styleCenter)
	if err != nil {
		fmt.Println("创建样式失败:", err)
		return
	}
	coloum := 1
	processNo := 1
	// cell, _ := excelize.CoordinatesToCellName(1, 1)
	f.SetCellValue(sheet, "A1", "Process Name")
	f.SetCellValue(sheet, "B1", "Duration")
	f.SetCellValue(sheet, "C1", "Start/End Time")
	for i, infoPtr := range oneBach {
		lineNo := processNo * (i + 1) * 2
		col_name := coloum
		m1, _ := excelize.CoordinatesToCellName(col_name, lineNo)
		m2, _ := excelize.CoordinatesToCellName(col_name, lineNo+1)
		f.MergeCell(sheet, m1, m2)

		// 为合并后的区域应用居中样式
		err = f.SetCellStyle(sheet, m1, m2, styleID)
		if err != nil {
			fmt.Println("设置样式失败:", err)
			return
		}

		cell, _ := excelize.CoordinatesToCellName(col_name, lineNo)
		f.SetCellValue(sheet, cell, infoPtr.ProcessName)

		col_duration := coloum + 1
		cell_du_1, _ := excelize.CoordinatesToCellName(col_duration, lineNo)
		cell_du_2, _ := excelize.CoordinatesToCellName(col_duration, lineNo+1)
		f.MergeCell(sheet, cell_du_1, cell_du_2)
		// 为合并后的区域应用居中样式
		err = f.SetCellStyle(sheet, cell_du_1, cell_du_2, styleID)
		if err != nil {
			fmt.Println("设置样式失败:", err)
			return
		}
		cell_du, _ := excelize.CoordinatesToCellName(col_duration, lineNo)
		f.SetCellValue(sheet, cell_du, infoPtr.Duration)

		col_start_bj := coloum + 2
		cell_0, _ := excelize.CoordinatesToCellName(col_start_bj, lineNo)
		f.SetCellValue(sheet, cell_0, infoPtr.StartTime)
		cell_1, _ := excelize.CoordinatesToCellName(col_start_bj, lineNo+1)
		f.SetCellValue(sheet, cell_1, infoPtr.EndTime)
	}
}

/**
* return each loop start slice
 */
func abstrct_every_startTime(processNames []string, allProcessInfoPtrSli []*ProcessInfo) []*ProcessInfo {
	startTimeBatchSli := make([]*ProcessInfo, 0)
	for range allProcessInfoPtrSli {
		var startProcessInfoPtr = allProcessInfoPtrSli[0]
		var index = 0
		for i, infoPtr := range allProcessInfoPtrSli {
			if infoPtr.StartTimeNoInt < startProcessInfoPtr.StartTimeNoInt {
				// 找到更早的启动时间
				startProcessInfoPtr = infoPtr
				index = i
			}
		}
		startTimeBatchSli = append(startTimeBatchSli, startProcessInfoPtr)
		allProcessInfoPtrSli = append(allProcessInfoPtrSli[:index], allProcessInfoPtrSli[index+1:]...)
	}
	return startTimeBatchSli
}

func compute_duration(allProcessInfoPtrSli []*ProcessInfo) {
	for _, processInfoPtr := range allProcessInfoPtrSli {
		StartTimeNoStr := strings.TrimSpace(processInfoPtr.StartTimeNoStr)
		EndTimeNoStr := strings.TrimSpace(processInfoPtr.EndTimeNoStr)
		if StartTimeNoStr != "" && EndTimeNoStr != "" {
			var start int64
			var end int64
			if _, err := fmt.Sscanf(StartTimeNoStr, "%d", &start); err != nil {
				zap.L().Error("转换StartTimeNo 时发生问题", zap.Error(err))
			}
			if _, err := fmt.Sscanf(EndTimeNoStr, "%d", &end); err != nil {
				zap.L().Error("转换 EndTimeNo 时发生问题", zap.Error(err))
			}
			processInfoPtr.StartTimeNoInt = start
			processInfoPtr.EndTimeNoInt = end
			nanose := end - start
			du := nanosecondsToHMSSS(nanose)
			processInfoPtr.Duration = du
		}
	}
}

func nanosecondsToHMSSS(nanoseconds int64) string {
	duration := time.Duration(nanoseconds)
	totalSeconds := int64(duration.Seconds())
	milliseconds := (nanoseconds % 1e9) / 1e6

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
}

func abstract_time(allProcessInfoPtrSli []*ProcessInfo) {
	for _, processInfoPtr := range allProcessInfoPtrSli {
		matchStartTime := re.FindStringSubmatch(processInfoPtr.StartLine)
		// 如果匹配成功，match[1] 是第一个子匹配（即括号内的内容）
		if len(matchStartTime) > 1 {
			processInfoPtr.StartTimeNoStr = matchStartTime[1]
			timeBj := ToBJTime(processInfoPtr.StartTimeNoStr)
			processInfoPtr.StartTime = timeBj
		} else {
			zap.L().Error("未找到匹配到日期的数字:" + processInfoPtr.StartLine)
		}

		matchEndTime := re.FindStringSubmatch(processInfoPtr.EndLine)
		if len(matchEndTime) > 1 {
			processInfoPtr.EndTimeNoStr = matchEndTime[1]
			timeBj := ToBJTime(processInfoPtr.EndTimeNoStr)
			processInfoPtr.EndTime = timeBj
		} else {
			fmt.Println("未找到匹配到日期的数字:" + processInfoPtr.EndLine)
		}
	}
}

func ToBJTime(num string) string {
	var ts int64
	n, err := fmt.Sscanf(num, "%d", &ts)
	if n != 1 || err != nil {
		return "" // 解析失败保留原内容
	}

	// 转换为北京时间
	t := time.Unix(0, ts).In(beijingLoc)
	return fmt.Sprintf("%s", t.Format("2006-01-02 15:04:05.000000000"))
}

func find_one_process(processName string, logLines []string) []*ProcessInfo {
	start := strings.ReplaceAll(startTpl, "XXXXXXXX", processName)
	end := strings.ReplaceAll(endTpl, "XXXXXXXX", processName)
	infoPtrSli := make([]*ProcessInfo, 0)
	haveStart := false
	haveEnd := false
	info := new(ProcessInfo)
	for _, logLine := range logLines {
		if haveStart && haveEnd {
			haveStart = false
			haveEnd = false
		}
		if strings.Contains(logLine, start) {
			haveStart = true
			info.StartLine = logLine
			// fmt.Println(start)
		}
		if strings.Contains(logLine, end) {
			haveEnd = true
			info.EndLine = logLine
			// fmt.Println(end)
		}
		if haveStart && haveEnd {
			info.ProcessName = processName
			infoPtrSli = append(infoPtrSli, info)
			info = new(ProcessInfo)
		}
	}
	return infoPtrSli
}

func load_log(inputFile string) []string {
	// 打开cupMonitor日志文件
	fin, err := os.Open(inputFile)
	if err != nil {
		zap.L().Error("无法打开日志文件", zap.String("inputFile", inputFile), zap.Error(err))
	}
	defer fin.Close()
	lines := make([]string, 0)
	scanner := bufio.NewScanner(fin)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}
	return lines
}

func load_process(process string) []string {
	// 打开进程名称文件
	fin, err := os.Open(process)
	if err != nil {
		zap.L().Error("无法打开进程名称文件", zap.String("process", process), zap.Error(err))
	}
	defer fin.Close()

	scanner := bufio.NewScanner(fin)
	names := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		names = append(names, line)
	}
	return names
}
