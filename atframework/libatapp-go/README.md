# libatapp-go

服务间通信层（golang）版本。
目标是和 <https://github.com/atframework/libatapp> 打通。

## TODO List

- [ ] app层管理和module结构
- [ ] 协议和配置管理
- [ ] connector抽象和接入层实现
  - [ ] libatbus-go connector
  - [ ] 本地回环 connector
  - [ ] endpoint管理和消息缓存
- [ ] etcd模块
  - [ ] 服务发现模块接入
- [ ] 策略路由
