const { default: MysqlBinlog } = require("../dist/binding.js");
const assert = require("assert");

assert(MysqlBinlog, "The expected module is undefined");

function testBasic()
{
    const instance = new MysqlBinlog({
        hostname: "localhost",
        port: 3306,
        username: "root",
        password: "password",
        tableRegexes: ['Users'],
    });
    instance.close();
}

function testInvalidParams()
{
    const instance = new MysqlBinlog();
    instance.close();
}

assert.throws(testBasic, undefined, "dial tcp [::1]:3306: connect: connection refused");
assert.throws(testInvalidParams, undefined, "testInvalidParams didn't throw");

console.log("Tests passed- everything looks OK!");