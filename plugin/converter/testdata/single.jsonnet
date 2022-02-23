{
   "kind": "pipeline",
   "name": "default",
   "steps": [
      {
         "commands": [
            "my_command"
         ],
         "image": "my_image",
         "name": "my_step"
      },
      {
         "commands": [
            "my_second_command1",
            "my_second_command2"
         ],
         "image": "my_second_image",
         "name": "my_second_step"
      },
      {
         "commands": [
            "my_third_command"
         ],
         "image": "my_third_image",
         "name": "my_third_step"
      }
   ],
   "type": "docker"
}
