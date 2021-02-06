namespace go codeRunner

include "base.thrift"

/*
Thrift version:
    0.13.0
generate thrift:
    thrift -r --gen go -out D:\WorkPlace\GoWorkPlace\thriftGen\src codeRunner.thrift
*/


service codeRunner extends base.baseService {
    base.CommomResp buildGo(),
    base.CommomResp buildCpp(),
    base.CommomResp run(),
}