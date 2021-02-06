namespace go baseService

// 注册服务到simpleServer需要先继承这个服务
service baseService {
    bool ping()
    bool close()    // 客户端主动断开连接
}

// 共用响应体
struct CommomResp {
    1: i32 status
    2: string msg
    3: binary payload
}
