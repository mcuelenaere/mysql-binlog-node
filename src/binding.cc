#include "mysql_binlog.h"

Napi::Object Init(Napi::Env env, Napi::Object exports) {
    MysqlBinlog::Init(env, exports);
    return exports;
}

NODE_API_MODULE(addon, Init)