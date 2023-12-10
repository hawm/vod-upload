1. 修改config.ini

2.执行程序

参数：
-spacename：空间名称
-filepath: 要上传的文件路径
-title: 上传后的标题，可以不设置，默认为文件名
-uploadpath: 上传后的路径，可以不设置，默认由服务端自动设置

vodupload -spacename=upload-test -filepath=./test.mp4


3.返回结果结构，使用英文逗号分隔

文件路径,上传后的标题,上传后的路径,是否上传成功,是否发布成功,错误信息

示例：
a.mp4,test,xxxxx.mp4,true,true,
b.mp4,test,,false,false,Something Wrong