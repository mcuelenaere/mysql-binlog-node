const MysqlBinlog = require("../dist/binding.js");
const assert = require("assert");

assert(MysqlBinlog, "The expected module is undefined");

function testBasic()
{
    const instance = new MysqlBinlog("localhost", 3306, "root", "password", ['Assets']);
    assert(instance.greet, "The expected method is not defined");
    assert.strictEqual(instance.greet("kermit"), "mr-yeoman", "Unexpected value returned");
}

function testInvalidParams()
{
    const instance = new MysqlBinlog();
}

assert.doesNotThrow(testBasic, undefined, "testBasic threw an expection");
assert.throws(testInvalidParams, undefined, "testInvalidParams didn't throw");

console.log("Tests passed- everything looks OK!");