{
  'targets': [
    {
        'target_name': 'mysql-binlog-go',
        'type': 'static_library',
        #'standalone_static_library': 1,
        'sources': ['<(INTERMEDIATE_DIR)/mysql-binlog-go.o'],
        'actions': [
            {
                'action_name': 'build',
                'inputs': ['main.go'],
                'outputs': [
                    '<(INTERMEDIATE_DIR)/mysql-binlog-go.o',
                    '<(INTERMEDIATE_DIR)/mysql-binlog-go.h',
                ],
                'action': ['go', 'build', '-buildmode', 'c-archive', '-o', '<(INTERMEDIATE_DIR)/mysql-binlog-go.o']
            }
        ],
        'copies': [
            {
                'destination': '<(PRODUCT_DIR)',
                'files': ['<(INTERMEDIATE_DIR)/mysql-binlog-go.h']
            }
        ]
    }
  ]
}