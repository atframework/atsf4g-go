package libatframe_utils_lang_utility

import "unsafe"

//  1. 手动定义 runtime.eface 结构
//     Go 源码里 intercafe 的真实定义
type eface struct {
	_type unsafe.Pointer // 类型信息
	data  unsafe.Pointer // 数据指针 <--- 你要找的就是这个
}

func init() {
	// --- 1. 大小检查 ---
	// interface{} 必须是两个机器字的大小 (16 bytes on 64-bit)
	// 如果 Go 将来加了字段，这里会报错
	if unsafe.Sizeof(eface{}) != unsafe.Sizeof(interface{}(nil)) {
		panic("CRITICAL: internal layout of interface{} size mismatch! Do not run this version.")
	}

	// --- 2. 字段布局/顺序检查 (金丝雀测试) ---

	// 定义一个“金丝雀”变量
	magicValue := 0xDEADBEEF

	// 把它的指针装进 interface{}
	// 注意：这里必须存指针 &magicValue，因为我们需要对比的是地址
	var i interface{} = &magicValue

	// 强转为我们的结构体
	myFace := (*eface)(unsafe.Pointer(&i))

	// 核心验证：
	// 如果 myFace.data 字段真的是数据指针，它应该等于 &magicValue
	// 如果 Go 偷偷把 _type 和 data 调换了位置，或者加了偏移，这里就会不相等
	if myFace.data != unsafe.Pointer(&magicValue) {
		panic("CRITICAL: internal layout of interface{} fields changed! 'data' is not at the expected offset.")
	}

	// 如果运行到这里没有 panic，说明当前环境是安全的
}

func GetDataPointerOfInterface(d interface{}) unsafe.Pointer {
	if d == nil {
		return nil
	}

	ifaceHeader := (*eface)(unsafe.Pointer(&d))
	return ifaceHeader.data
}
