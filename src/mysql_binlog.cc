#include "mysql_binlog.h"
#include <iostream>

using namespace Napi;

const size_t MAX_TSFN_QUEUE_SIZE = 1000;

static std::vector<std::string> toStringVec(const Napi::Array& array) {
    std::vector<std::string> string_vec;
    for (const auto& e : array) {
        string_vec.push_back(static_cast<Napi::Value>(e.second).As<Napi::String>().Utf8Value());
    }
    return string_vec;
}

static std::vector<const char*> toCharVec(const std::vector<std::string>& array) {
    std::vector<const char*> char_vec;
    for (const auto& s : array) {
        char_vec.push_back(s.c_str());
    }
    return char_vec;
}

MysqlBinlog::MysqlBinlog(const Napi::CallbackInfo& info) : ObjectWrap(info) {
    Napi::Env env = info.Env();

    if (info.Length() < 6) {
        Napi::TypeError::New(env, "Wrong number of arguments").ThrowAsJavaScriptException();
        return;
    }

    if (
        !info[0].IsString() ||
        !info[1].IsNumber() ||
        !info[2].IsString() ||
        !info[3].IsString() ||
        !info[4].IsArray() ||
        !info[5].IsFunction()
    ) {
        Napi::TypeError::New(env, "Wrong argument type(s)").ThrowAsJavaScriptException();
        return;
    }

    auto hostname = info[0].As<Napi::String>().Utf8Value();
    auto port = info[1].As<Napi::Number>().Int32Value();
    auto username = info[2].As<Napi::String>().Utf8Value();
    auto password = info[3].As<Napi::String>().Utf8Value();
    auto tableRegexes = toStringVec(info[4].As<Napi::Array>());
    auto cTableRegexes = toCharVec(tableRegexes);

    this->callback = TSF::New(
        env,
        info[5].As<Napi::Function>(),
        "MysqlBinlog callback",
        MAX_TSFN_QUEUE_SIZE,
        1,
        nullptr,
        [](Napi::Env, void*, std::nullptr_t*) {
            // do nothing
        }
    );

    BinlogPosition *binlogPosition;
    if (info.Length() == 7 && info[6].IsObject()) {
        auto jsPos = info[6].As<Napi::Object>();
        auto name = jsPos.Get("name").As<Napi::String>().Utf8Value();
        auto pos = jsPos.Get("position").As<Napi::Number>().Uint32Value();

        BinlogPosition position = {
            .name = name.c_str(),
            .position = pos,
        };
        binlogPosition = &position;
    } else {
        binlogPosition = nullptr;
    }

    char* error = nullptr;
    this->go_handle = Binlog_New({
       .hostname = hostname.c_str(),
       .port = static_cast<int16_t>(port),
       .username = username.c_str(),
       .password = password.c_str(),
       .table_regexes = &cTableRegexes[0],
       .table_regexes_count = cTableRegexes.size(),
       .binlog_position = binlogPosition,
       .callback = [](auto user_data, auto event) {
            auto self = static_cast<MysqlBinlog*>(user_data);
            self->OnEvent(event);
       },
       .user_data = this,
    }, &error);
    if (this->go_handle == 0) {
        if (error != nullptr) {
            Napi::Error::New(env, error).ThrowAsJavaScriptException();
            free(error);
        } else {
            Napi::Error::New(env, "unknown error in Go land occurred").ThrowAsJavaScriptException();
        }
        this->callback.Release();
        return;
    }
}

MysqlBinlog::~MysqlBinlog() {
    this->Release();
}

void MysqlBinlog::OnEvent(const char* event) {
    auto status = this->callback.Acquire();
    if (status != napi_ok) {
        free((void*) event);
        return;
    }

    status = this->callback.BlockingCall(const_cast<char*>(event));
    if (status != napi_ok) {
        free((void*) event);
        // passthrough, make sure we lower the refcount with 1
    }

    status = this->callback.Release();
    if (status != napi_ok) {
        // do nothing, `event` was sent over to the queue, so it'll be freed in `CallJs`
    }
}

void CallJs(Napi::Env env, Napi::Function callback, std::nullptr_t* context, char* event) {
    if (env != nullptr && callback != nullptr && event != nullptr) {
        callback.Call(env.Null(), {Napi::String::New(env, event)});
    }
    if (event != nullptr) {
        free((void*) event);
    }
}

void MysqlBinlog::Release() {
    if (this->go_handle != 0) {
        Binlog_Close(this->go_handle);
        Binlog_Free(this->go_handle);
        this->go_handle = 0;
        this->callback.Release();
    }
}

Napi::Value MysqlBinlog::Close(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();

    this->Release();

    return env.Undefined();
}

Napi::Value SetLogger(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();

    // TODO

    return env.Undefined();
}

void MysqlBinlog::Init(Napi::Env env, Napi::Object exports) {
    auto klass = DefineClass(env, "MysqlBinlog", {
        MysqlBinlog::InstanceMethod("close", &MysqlBinlog::Close),
    });
    exports.Set("MysqlBinlog", klass);
    exports.Set("setLogger", Napi::Function::New(env, SetLogger));
}