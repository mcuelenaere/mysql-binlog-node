#pragma once

#include <stdint.h>
#include <napi.h>
#include "mysql-binlog-go.h"

void CallJs(Napi::Env, Napi::Function, nullptr_t*, char*);
using TSF = Napi::TypedThreadSafeFunction<nullptr_t, char, CallJs>;

class MysqlBinlog : public Napi::ObjectWrap<MysqlBinlog>
{
public:
    MysqlBinlog(const Napi::CallbackInfo&);
    ~MysqlBinlog();
    Napi::Value Close(const Napi::CallbackInfo&);

    static void Init(Napi::Env, Napi::Object);

private:
    void Release();
    void OnEvent(const char*);

    uintptr_t go_handle;
    TSF callback;
};
