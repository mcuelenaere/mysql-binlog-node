const { default: MysqlBinlog } = require("../dist/binding.js");

describe('MysqlBinlog', () => {
    test('should throw error when creating instance with invalid parameters', async () => {
        expect(MysqlBinlog).toBeDefined();

        await expect(MysqlBinlog.create()).rejects.toThrow();
    });

    test('should throw connection error when connecting to invalid host', async () => {
        expect(MysqlBinlog).toBeDefined();

        await expect(MysqlBinlog.create({
            hostname: "localhost",
            port: 3306,
            username: "root",
            password: "password",
            tableRegexes: ['Users'],
        })).rejects.toThrow(/dial tcp \[::1\]:3306: (connect|connectex).*refused/i);
    });
});