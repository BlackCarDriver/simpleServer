namespace go codeRunner

struct CommomResp {
    1: i32 status
    2: string msg
    3: binary payload
}

service codeRunner {
    bool ping()
}