{
  'targets': [
    {
        'target_name': 'mysql-binlog-go',
        'type': 'static_library',
        #'standalone_static_library': 1,
        'sources': ['<(INTERMEDIATE_DIR)/mysql-binlog-go.<(lib_suffix)'],
        'actions': [
            {
                'action_name': 'build',
                'inputs': ['main.go'],
                'outputs': [
                    '<(INTERMEDIATE_DIR)/mysql-binlog-go.<(lib_suffix)',
                    '<(INTERMEDIATE_DIR)/mysql-binlog-go.h',
                ],
                'action': ['node', 'build.js', '<(INTERMEDIATE_DIR)/mysql-binlog-go.<(lib_suffix)']
            }
        ],
        'copies': [
            {
                'destination': '<(PRODUCT_DIR)',
                'files': ['<(INTERMEDIATE_DIR)/mysql-binlog-go.h']
            }
        ]
    }
  ],
  'conditions': [
    ['OS=="mac"', {'variables': {'lib_suffix': 'o'}}],
    ['OS=="win"', {'variables': {'lib_suffix': 'lib'}}],
    ['OS=="linux"', {'variables': {'lib_suffix': 'a'}}],
  ]
}