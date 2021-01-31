namespace go codeRunner

/*
Thrift version:
    0.13.0
generate thrift:
    thrift -r --gen go codeRunner.thrift
*/

// 注册服务到simpleServer需要先继承这个服务
service baseService {
    bool ping()
}

// 共用响应体
struct CommomResp {
    1: i32 status
    2: string msg
    3: binary payload
}

service codeRunner extends baseService {
    CommomResp buildGo(),
    CommomResp buildCpp(),
    CommomResp run(),
}