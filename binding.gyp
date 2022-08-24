{
  'targets': [
    {
      'target_name': 'mysql-binlog-native',
      'sources': [ 'src/binding.cc', 'src/mysql_binlog.cc' ],
      'include_dirs': [
        "<!@(node -p \"require('node-addon-api').include\")",
        '<(PRODUCT_DIR)'
      ],
      'dependencies': [
        "<!(node -p \"require('node-addon-api').gyp\")",
        'golang/build.gyp:mysql-binlog-go'
      ],
      'cflags!': [ '-fno-exceptions' ],
      'cflags_cc!': [ '-fno-exceptions' ],
      'xcode_settings': {
        'GCC_ENABLE_CPP_EXCEPTIONS': 'YES',
        'CLANG_CXX_LIBRARY': 'libc++',
        'MACOSX_DEPLOYMENT_TARGET': '10.7'
      },
      'msvs_settings': {
        'VCCLCompilerTool': { 'ExceptionHandling': 1 },
      }
    }
  ]
}