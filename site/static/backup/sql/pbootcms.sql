-- Online Database Management SQL Dump
-- 数据库名: pbootcms
-- 生成日期: 2020-07-05 23:55:34
-- PHP 版本: 5.6.33

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
SET time_zone = "+08:00";
SET NAMES utf8;

-- --------------------------------------------------------

--
-- 表的结构 `ay_area`
--

DROP TABLE IF EXISTS `ay_area`;
CREATE TABLE `ay_area` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '区域编号',
 `acode` varchar(20) NOT NULL COMMENT '区域编码',
 `pcode` varchar(20) NOT NULL COMMENT '区域父编码',
 `name` varchar(50) NOT NULL COMMENT '区域名称',
 `domain` varchar(100) NOT NULL COMMENT '区域绑定域名',
 `is_default` char(1) NOT NULL DEFAULT '0' COMMENT '是否默认',
 `create_user` varchar(30) NOT NULL COMMENT '添加人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '添加时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 PRIMARY KEY (`id`),
 UNIQUE KEY `ay_area_acode` (`acode`),
 KEY `ay_area_pcode` (`pcode`)
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_area`
--

INSERT INTO `ay_area` (`id`,`acode`,`pcode`,`name`,`domain`,`is_default`,`create_user`,`update_user`,`create_time`,`update_time`) VALUES
('1','cn','0','中文','','1','admin','admin','2017-11-30 13:55:37','2018-04-13 11:40:49');

-- --------------------------------------------------------

--
-- 表的结构 `ay_company`
--

DROP TABLE IF EXISTS `ay_company`;
CREATE TABLE `ay_company` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '站点编号',
 `acode` varchar(20) NOT NULL COMMENT '区域代码',
 `name` varchar(100) NOT NULL COMMENT '公司名称',
 `address` varchar(200) NOT NULL COMMENT '公司地址',
 `postcode` varchar(6) NOT NULL COMMENT '邮政编码',
 `contact` varchar(10) NOT NULL COMMENT '公司联系人',
 `mobile` varchar(50) NOT NULL COMMENT '手机号码',
 `phone` varchar(50) NOT NULL COMMENT '电话号码',
 `fax` varchar(50) NOT NULL COMMENT '公司传真',
 `email` varchar(30) NOT NULL COMMENT '电子邮箱',
 `qq` varchar(50) NOT NULL COMMENT '公司QQ',
 `weixin` varchar(100) NOT NULL COMMENT '微信图标',
 `blicense` varchar(20) NOT NULL COMMENT '营业执照代码',
 `other` varchar(200) NOT NULL COMMENT '其他信息',
 PRIMARY KEY (`id`),
 KEY `ay_company_acode` (`acode`)
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_company`
--

INSERT INTO `ay_company` (`id`,`acode`,`name`,`address`,`postcode`,`contact`,`mobile`,`phone`,`fax`,`email`,`qq`,`weixin`,`blicense`,`other`) VALUES
('1','cn','PbootCMS-Go','','','','','','','','','','','');

-- --------------------------------------------------------

--
-- 表的结构 `ay_config`
--

DROP TABLE IF EXISTS `ay_config`;
CREATE TABLE `ay_config` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `name` varchar(30) NOT NULL COMMENT '名称',
 `value` varchar(200) NOT NULL COMMENT '值',
 `type` char(1) NOT NULL DEFAULT '1' COMMENT '配置类型',
 `sorting` int(10) unsigned NOT NULL DEFAULT '255' COMMENT '排序',
 `description` varchar(30) NOT NULL COMMENT '描述文本',
 PRIMARY KEY (`id`),
 KEY `ay_config_name` (`name`)
) ENGINE=MyISAM AUTO_INCREMENT=45 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_config`
--

INSERT INTO `ay_config` (`id`,`name`,`value`,`type`,`sorting`,`description`) VALUES
('1','open_wap','0','1','255','手机版'),
('2','message_check_code','1','1','255','留言验证码'),
('3','smtp_server','smtp.qq.com','2','255','邮件SMTP服务器'),
('4','smtp_port','465','2','255','邮件SMTP端口'),
('5','smtp_ssl','1','1','255','邮件是否安全连接'),
('6','smtp_username','','2','255','邮件发送账号'),
('7','smtp_password','','2','255','邮件发送密码'),
('8','admin_check_code','1','1','255','后台验证码'),
('9','weixin_appid','','2','255','微信APPID'),
('10','weixin_secret','','2','255','微信SECRET'),
('11','message_send_mail','0','1','255','留言发送邮件开关'),
('12','message_send_to','','1','255','留言发送到邮箱'),
('13','api_open','0','2','255','API开关'),
('14','api_auth','1','2','255','API强制认证'),
('15','api_appid','','2','255','API认证用户'),
('16','api_secret','','2','255','API认证密钥'),
('17','baidu_zz_token','','2','255','百度站长密钥'),
('18','baidu_xzh_appid','','2','255','熊掌号appid'),
('19','baidu_xzh_token','','2','255','熊掌号token'),
('20','wap_domain','','2','255','手机绑定域名'),
('21','gzip','0','2','255','GZIP压缩'),
('22','content_tags_replace_num','','2','255','内容关键字替换次数'),
('23','smtp_username_test','','2','255','测试邮箱'),
('24','form_send_mail','0','2','255','表单发送邮件'),
('25','baidu_xzh_type','0','2','255','熊掌号推送类型'),
('26','watermark_open','0','2','255','水印开关'),
('27','watermark_text','PbootCMS-Go','2','255','水印文本'),
('28','watermark_text_font','','2','255','水印文本字体'),
('29','watermark_text_size','20','2','255','水印文本字号'),
('30','watermark_text_color','100,100,100','2','255','水印文本字体颜色'),
('31','watermark_pic','','2','255','水印图片'),
('32','watermark_position','4','2','255','水印位置'),
('33','message_verify','1','2','255','留言审核'),
('34','form_check_code','1','2','255','表单验证码'),
('35','lock_count','5','2','255','登陆锁定阈值'),
('36','lock_time','900','2','255','登录锁定时间'),
('37','url_rule_type','3','2','255','路径类型'),
('38','message_status','0','2','255','留言开关'),
('39','form_status','0','2','255','表单开关'),
('40','tpl_html_dir','html','2','255','模板HTML目录'),
('41','ip_deny','','2','255','IP黑名单'),
('42','ip_allow','','2','255','IP白名单'),
('43','url_index_404','0','2','255','跳转404'),
('44','tpl_html_cache','1','1','255','动态缓存开关'),
('45','tpl_html_cache_time','900000000000','2','255','缓存有效期(秒)'),
('46','register_status','0','2','255','会员注册'),
('47','login_status','0','2','255','会员登录'),
('48','comment_status','0','2','255','评论开关'),
('49','comment_anonymous','0','2','255','匿名评论'),
('50','comment_check_code','1','2','255','评论验证码'),
('51','comment_verify','1','2','255','评论审核'),
('52','comment_send_mail','0','2','255','评论发邮件'),
('53','register_check_code','1','2','255','注册验证码'),
('54','login_check_code','1','2','255','登录验证码'),
('55','password_strong','1','2','255','强密码'),
('56','llms_open','0','2','255','llms.txt'),
('57','spiderlog','0','2','255','蜘蛛日志'),
('58','message_rqlogin','0','2','255','留言需登录');

-- --------------------------------------------------------

--
-- 表的结构 `ay_content`
--

DROP TABLE IF EXISTS `ay_content`;
CREATE TABLE `ay_content` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `acode` varchar(20) NOT NULL COMMENT '区域',
 `scode` varchar(20) NOT NULL COMMENT '内容栏目',
 `subscode` varchar(20) NOT NULL COMMENT '副栏目',
 `title` varchar(100) NOT NULL COMMENT '标题',
 `titlecolor` varchar(7) NOT NULL COMMENT '标题颜色',
 `subtitle` varchar(100) NOT NULL COMMENT '副标题',
 `filename` varchar(50) NOT NULL COMMENT '自定义文件名',
 `author` varchar(30) NOT NULL COMMENT '作者',
 `source` varchar(30) NOT NULL COMMENT '来源',
 `outlink` varchar(100) NOT NULL COMMENT '外链地址',
 `date` datetime NOT NULL COMMENT '发布日期',
 `ico` varchar(100) NOT NULL COMMENT '缩略图',
 `pics` varchar(1000) NOT NULL COMMENT '多图片',
 `picstitle` varchar(1000) NOT NULL COMMENT '多图片标题',
 `content` mediumtext NOT NULL COMMENT '内容',
 `tags` varchar(500) NOT NULL COMMENT 'tag关键字',
 `enclosure` varchar(100) NOT NULL COMMENT '附件',
 `keywords` varchar(200) NOT NULL COMMENT '关键字',
 `description` varchar(500) NOT NULL COMMENT '描述',
 `sorting` int(10) unsigned NOT NULL DEFAULT '255' COMMENT '内容排序',
 `status` char(1) NOT NULL DEFAULT '1' COMMENT '状态',
 `istop` char(1) NOT NULL DEFAULT '0' COMMENT '是否置顶',
 `isrecommend` char(1) NOT NULL DEFAULT '0' COMMENT '是否推荐',
 `isheadline` char(1) NOT NULL DEFAULT '0' COMMENT '是否头条',
 `visits` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '访问数',
 `likes` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '点赞数',
 `oppose` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '反对数',
 `create_user` varchar(30) NOT NULL COMMENT '创建人员',
 `update_user` varchar(20) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '创建时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 `gtype` char(1) NOT NULL DEFAULT '4',
 `gid` varchar(20) NOT NULL DEFAULT '',
 `gnote` varchar(100) NOT NULL DEFAULT '',
 PRIMARY KEY (`id`),
 KEY `ay_content_scode` (`scode`),
 KEY `ay_content_subscode` (`subscode`),
 KEY `ay_content_acode` (`acode`),
 KEY `ay_content_filename` (`filename`),
 KEY `ay_content_date` (`date`),
 KEY `ay_content_sorting` (`sorting`),
 KEY `ay_content_status` (`status`),
 KEY `ay_content_title_index` (`title`)
) ENGINE=MyISAM AUTO_INCREMENT=18 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_content`
--

-- cleared demo -- cleared demo INSERT INTO `ay_content`;
-- --------------------------------------------------------

--
-- 表的结构 `ay_content_ext`
--

DROP TABLE IF EXISTS `ay_content_ext`;
CREATE TABLE `ay_content_ext` (
 `extid` int(10) unsigned NOT NULL AUTO_INCREMENT,
 `contentid` int(10) unsigned NOT NULL,
 `ext_price` varchar(100) DEFAULT NULL COMMENT '产品价格',
 `ext_type` varchar(100) DEFAULT NULL COMMENT '类型',
 `ext_color` varchar(100) DEFAULT NULL COMMENT '颜色',
 PRIMARY KEY (`extid`),
 KEY `ay_content_ext_contentid` (`contentid`),
 KEY ay_content_ext_color_index (`ext_color`),
 KEY ay_content_ext_price_index (`ext_price`),
 KEY ay_content_ext_type_index (`ext_type`)
) ENGINE=MyISAM AUTO_INCREMENT=6 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_content_ext`
--

-- cleared demo -- cleared demo INSERT INTO `ay_content_ext`;
-- --------------------------------------------------------

--
-- 表的结构 `ay_content_sort`
--

DROP TABLE IF EXISTS `ay_content_sort`;
CREATE TABLE `ay_content_sort` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `acode` varchar(20) NOT NULL COMMENT '区域编码',
 `mcode` varchar(20) NOT NULL COMMENT '内容模型编码',
 `pcode` varchar(20) NOT NULL COMMENT '父编码',
 `scode` varchar(20) NOT NULL COMMENT '分类编码',
 `name` varchar(100) NOT NULL COMMENT '分类名称',
 `listtpl` varchar(50) NOT NULL COMMENT '列表页模板',
 `contenttpl` varchar(50) NOT NULL COMMENT '内容页模板',
 `status` char(1) NOT NULL DEFAULT '1' COMMENT '状态',
 `outlink` varchar(100) NOT NULL COMMENT '转外链接',
 `subname` varchar(200) NOT NULL COMMENT '附加名称',
 `def1` varchar(1000) NOT NULL COMMENT '栏目描述1',
 `def2` varchar(1000) NOT NULL COMMENT '栏目描述2',
 `def3` varchar(1000) NOT NULL COMMENT '栏目描述3',
 `ico` varchar(100) NOT NULL COMMENT '分类缩略图',
 `pic` varchar(100) NOT NULL COMMENT '分类大图',
 `title` varchar(100) NOT NULL COMMENT 'seo标题',
 `keywords` varchar(200) NOT NULL COMMENT '分类关键字',
 `description` varchar(500) NOT NULL COMMENT '分类描述',
 `filename` varchar(30) NOT NULL COMMENT '自定义文件名',
 `sorting` int(10) unsigned NOT NULL DEFAULT '255' COMMENT '排序',
 `create_user` varchar(30) NOT NULL COMMENT '创建人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '创建时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 `gtype` char(1) NOT NULL DEFAULT '4',
 `gid` varchar(20) NOT NULL DEFAULT '',
 `gnote` varchar(100) NOT NULL DEFAULT '',
 PRIMARY KEY (`id`),
 UNIQUE KEY `ay_content_sort_scode` (`scode`),
 KEY `ay_content_sort_pcode` (`pcode`),
 KEY `ay_content_sort_acode` (`acode`),
 KEY `ay_content_sort_mcode` (`mcode`),
 KEY `ay_content_sort_filename` (`filename`),
 KEY `ay_content_sort_sorting` (`sorting`)
) ENGINE=MyISAM AUTO_INCREMENT=12 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_content_sort`
--

-- cleared demo -- cleared demo INSERT INTO `ay_content_sort`;
-- --------------------------------------------------------

--
-- 表的结构 `ay_diy_telephone`
--

DROP TABLE IF EXISTS `ay_diy_telephone`;
CREATE TABLE `ay_diy_telephone` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
 `create_time` datetime NOT NULL,
 `tel` varchar(20) DEFAULT NULL COMMENT '电话号码',
 PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- 表的结构 `ay_extfield`
--

DROP TABLE IF EXISTS `ay_extfield`;
CREATE TABLE `ay_extfield` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `mcode` varchar(20) NOT NULL COMMENT '模型编码',
 `name` varchar(30) NOT NULL COMMENT '字段名称',
 `type` char(2) NOT NULL COMMENT '字段类型',
 `value` varchar(500) NOT NULL COMMENT '单选或多选值',
 `description` varchar(30) NOT NULL COMMENT '描述文本',
 `sorting` int(11) NOT NULL COMMENT '排序',
 PRIMARY KEY (`id`),
 KEY `ay_extfield_mcode` (`mcode`)
) ENGINE=MyISAM AUTO_INCREMENT=4 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_extfield`
--

INSERT INTO `ay_extfield` (`id`,`mcode`,`name`,`type`,`value`,`description`,`sorting`) VALUES
('1','3','ext_price','1','','产品价格','255'),
('2','3','ext_type','4','基础版,专业版,旗舰版','类型','255'),
('3','3','ext_color','4','红色,橙色,黄色,绿色,蓝色,紫色','颜色','255');

-- --------------------------------------------------------

--
-- 表的结构 `ay_form`
--

DROP TABLE IF EXISTS `ay_form`;
CREATE TABLE `ay_form` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `fcode` varchar(20) NOT NULL COMMENT '表单编码',
 `form_name` varchar(30) NOT NULL COMMENT '表单名称',
 `table_name` varchar(30) NOT NULL COMMENT '表名称',
 `create_user` varchar(30) NOT NULL COMMENT '添加人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '添加时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 PRIMARY KEY (`id`),
 UNIQUE KEY `ay_form_fcode` (`fcode`)
) ENGINE=MyISAM AUTO_INCREMENT=3 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_form`
--

INSERT INTO `ay_form` (`id`,`fcode`,`form_name`,`table_name`,`create_user`,`update_user`,`create_time`,`update_time`) VALUES
('1','1','在线留言','ay_message','admin','admin','2018-04-11 17:31:29','2018-04-11 17:31:29'),
('2','2','搜集电话','ay_diy_telephone','admin','admin','2018-11-30 15:17:40','2018-11-30 15:17:40');

-- --------------------------------------------------------

--
-- 表的结构 `ay_form_field`
--

DROP TABLE IF EXISTS `ay_form_field`;
CREATE TABLE `ay_form_field` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `fcode` varchar(20) NOT NULL COMMENT '表单编码',
 `name` varchar(30) NOT NULL COMMENT '字段名称',
 `length` int(10) unsigned NOT NULL COMMENT '字段长度',
 `required` char(1) NOT NULL DEFAULT '0' COMMENT '是否必填',
 `description` varchar(30) NOT NULL COMMENT '描述文本',
 `sorting` int(10) unsigned NOT NULL DEFAULT '255' COMMENT '排序',
 `create_user` varchar(30) NOT NULL COMMENT '添加人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '添加时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 PRIMARY KEY (`id`),
 KEY `ay_form_field_fcode` (`fcode`),
 KEY `ay_form_field_sorting` (`sorting`)
) ENGINE=MyISAM AUTO_INCREMENT=5 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_form_field`
--

INSERT INTO `ay_form_field` (`id`,`fcode`,`name`,`length`,`required`,`description`,`sorting`,`create_user`,`update_user`,`create_time`,`update_time`) VALUES
('1','1','contacts','10','1','联系人','255','admin','admin','2018-07-14 18:24:02','2018-07-15 17:47:43'),
('2','1','mobile','12','1','手机','255','admin','admin','2018-07-14 18:24:02','2018-07-15 17:47:44'),
('3','1','content','500','1','内容','255','admin','admin','2018-07-14 18:24:02','2018-07-15 17:47:45'),
('4','2','tel','20','1','电话号码','255','admin','admin','2018-11-30 15:18:00','2018-11-30 15:18:00');

-- --------------------------------------------------------

--
-- 表的结构 `ay_label`
--

DROP TABLE IF EXISTS `ay_label`;
CREATE TABLE `ay_label` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `name` varchar(100) NOT NULL COMMENT '名称',
 `value` varchar(500) NOT NULL COMMENT '值',
 `type` char(1) NOT NULL DEFAULT '1' COMMENT '字段类型',
 `description` varchar(30) NOT NULL COMMENT '描述',
 `create_user` varchar(30) NOT NULL COMMENT '创建人员',
 `update_user` varchar(20) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '创建时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 PRIMARY KEY (`id`)
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_label`
--

-- cleared demo -- cleared demo INSERT INTO `ay_label`;
-- --------------------------------------------------------

--
-- 表的结构 `ay_link`
--

DROP TABLE IF EXISTS `ay_link`;
CREATE TABLE `ay_link` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '序号',
 `acode` varchar(20) NOT NULL COMMENT '区域编码',
 `gid` int(10) unsigned NOT NULL COMMENT '分组序号',
 `name` varchar(50) NOT NULL COMMENT '链接名称',
 `link` varchar(100) NOT NULL COMMENT '跳转链接',
 `logo` varchar(100) NOT NULL COMMENT '图片地址',
 `sorting` int(11) NOT NULL COMMENT '排序',
 `create_user` varchar(30) NOT NULL COMMENT '创建人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '创建时间',
 `update_time` datetime NOT NULL COMMENT '修改时间',
 PRIMARY KEY (`id`),
 KEY `ay_link_acode` (`acode`),
 KEY `ay_link_gid` (`gid`),
 KEY `ay_link_sorting` (`sorting`)
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_link`
--

-- cleared demo -- cleared demo INSERT INTO `ay_link`;
-- --------------------------------------------------------

--
-- 表的结构 `ay_member`
--

DROP TABLE IF EXISTS `ay_member`;
CREATE TABLE `ay_member` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
 `ucode` varchar(20) NOT NULL,
 `username` varchar(100) NOT NULL,
 `useremail` varchar(50) NOT NULL DEFAULT '',
 `usermobile` varchar(11) NOT NULL DEFAULT '',
 `nickname` varchar(100) NOT NULL,
 `password` varchar(32) NOT NULL,
 `headpic` varchar(200) NOT NULL,
 `status` char(1) NOT NULL,
 `activation` char(1) NOT NULL DEFAULT '1',
 `gid` varchar(20) NOT NULL,
 `wxid` varchar(50) NOT NULL,
 `qqid` varchar(50) NOT NULL,
 `wbid` varchar(50) NOT NULL,
 `score` int(10) unsigned NOT NULL DEFAULT '0',
 `register_time` datetime NOT NULL,
 `login_count` int(10) unsigned NOT NULL DEFAULT '0',
 `last_login_ip` varchar(11) NOT NULL,
 `last_login_time` varchar(11) NOT NULL,
 `sex` varchar(2) NOT NULL DEFAULT '',
 `birthday` varchar(20) NOT NULL DEFAULT '',
 `qq` varchar(15) NOT NULL DEFAULT '',
 PRIMARY KEY (`id`),
 UNIQUE KEY `ay_member_ucode` (`ucode`),
 UNIQUE KEY `ay_member_username` (`username`),
 KEY `ay_member_gid` (`gid`),
 KEY `ay_member_wxid` (`wxid`),
 KEY `ay_member_qqid` (`qqid`),
 KEY `ay_member_wbid` (`wbid`),
 KEY `ay_member_useremail` (`useremail`),
 KEY `ay_member_usermobile` (`usermobile`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- 表的结构 `ay_member_comment`
--

DROP TABLE IF EXISTS `ay_member_comment`;
CREATE TABLE `ay_member_comment` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
 `pid` int(10) unsigned NOT NULL DEFAULT '0',
 `contentid` int(10) unsigned NOT NULL,
 `comment` varchar(1000) NOT NULL,
 `uid` int(10) unsigned NOT NULL,
 `puid` int(10) unsigned NOT NULL,
 `likes` int(10) unsigned NOT NULL DEFAULT '0',
 `oppose` int(10) unsigned NOT NULL DEFAULT '0',
 `status` char(1) NOT NULL,
 `user_ip` varchar(11) NOT NULL,
 `user_os` varchar(30) NOT NULL,
 `user_bs` varchar(30) NOT NULL,
 `create_time` datetime NOT NULL,
 `update_user` varchar(30) NOT NULL,
 `update_time` datetime NOT NULL,
 PRIMARY KEY (`id`),
 KEY `ay_member_comment_pid` (`pid`),
 KEY `ay_member_comment_contentid` (`contentid`),
 KEY `ay_member_comment_uid` (`uid`),
 KEY `ay_member_comment_puid` (`puid`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- 表的结构 `ay_member_field`
--

DROP TABLE IF EXISTS `ay_member_field`;
CREATE TABLE `ay_member_field` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
 `name` varchar(30) NOT NULL,
 `length` int(10) unsigned NOT NULL,
 `required` char(1) NOT NULL,
 `description` varchar(30) NOT NULL,
 `sorting` int(10) unsigned NOT NULL,
 `status` char(1) NOT NULL,
 `create_user` varchar(30) NOT NULL,
 `update_user` varchar(30) NOT NULL,
 `create_time` datetime NOT NULL,
 `update_time` datetime NOT NULL,
 PRIMARY KEY (`id`)
) ENGINE=MyISAM AUTO_INCREMENT=4 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_member_field`
--

INSERT INTO `ay_member_field` (`id`,`name`,`length`,`required`,`description`,`sorting`,`status`,`create_user`,`update_user`,`create_time`,`update_time`) VALUES
('1','sex','2','0','性别','255','1','admin','admin','2020-06-25 00:00:00','2020-06-25 00:00:00'),
('2','birthday','20','0','生日','255','1','admin','admin','2020-06-25 00:00:00','2020-06-25 00:00:00'),
('3','qq','15','0','QQ','255','1','admin','admin','2020-06-25 00:00:00','2020-06-25 00:00:00');

-- --------------------------------------------------------

--
-- 表的结构 `ay_member_group`
--

DROP TABLE IF EXISTS `ay_member_group`;
CREATE TABLE `ay_member_group` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
 `gcode` varchar(20) NOT NULL,
 `gname` varchar(100) NOT NULL,
 `description` varchar(200) NOT NULL,
 `status` varchar(1) NOT NULL,
 `lscore` int(10) unsigned NOT NULL DEFAULT '0',
 `uscore` int(10) unsigned NOT NULL DEFAULT '0',
 `create_user` varchar(30) NOT NULL,
 `update_user` varchar(30) NOT NULL,
 `create_time` datetime NOT NULL,
 `update_time` datetime NOT NULL,
 PRIMARY KEY (`id`),
 UNIQUE KEY `ay_member_group_gcode` (`gcode`)
) ENGINE=MyISAM AUTO_INCREMENT=4 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_member_group`
--

INSERT INTO `ay_member_group` (`id`,`gcode`,`gname`,`description`,`status`,`lscore`,`uscore`,`create_user`,`update_user`,`create_time`,`update_time`) VALUES
('1','1','初级会员','初级会员具备基本的权限','1','0','999','admin','admin','2020-06-25 00:00:00','2020-06-25 00:00:00'),
('2','2','中级会员','中级会员具备部分特殊权限','1','1000','9999','admin','admin','2020-06-25 00:00:00','2020-06-25 00:00:00'),
('3','3','高级会员','高级会员具备全部特殊权限','1','10000','4294967295','admin','admin','2020-06-25 00:00:00','2020-06-25 00:00:00');

-- --------------------------------------------------------

--
-- 表的结构 `ay_menu`
--

DROP TABLE IF EXISTS `ay_menu`;
CREATE TABLE `ay_menu` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '菜单编号',
 `mcode` varchar(20) NOT NULL COMMENT '菜单编码',
 `pcode` varchar(20) NOT NULL COMMENT '上级菜单',
 `name` varchar(50) NOT NULL COMMENT '菜单名称',
 `url` varchar(100) NOT NULL COMMENT '菜单地址',
 `sorting` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '菜单排序',
 `status` char(1) NOT NULL DEFAULT '1' COMMENT '是否启用',
 `shortcut` char(1) NOT NULL DEFAULT '0' COMMENT '桌面图标',
 `ico` varchar(30) NOT NULL COMMENT '菜单图标',
 `create_user` varchar(30) NOT NULL COMMENT '创建人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '创建时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 PRIMARY KEY (`id`),
 UNIQUE KEY `ay_menu_mcode` (`mcode`),
 KEY `ay_menu_pcode` (`pcode`),
 KEY `ay_menu_sorting` (`sorting`)
) ENGINE=MyISAM AUTO_INCREMENT=67 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_menu`
--

INSERT INTO `ay_menu` (`id`,`mcode`,`pcode`,`name`,`url`,`sorting`,`status`,`shortcut`,`ico`,`create_user`,`update_user`,`create_time`,`update_time`) VALUES
('1','M101','0','系统管理','/admin/M101/index','900','1','0','fa-cog','admin','admin','0000-00-00 00:00:00','2018-04-30 14:52:57'),
('2','M102','M101','数据区域','/admin/Area/index','901','1','1','fa-sitemap','admin','admin','0000-00-00 00:00:00','2018-04-30 14:54:23'),
('3','M103','M101','系统菜单','/admin/Menu/index','902','0','0','fa-bars','admin','admin','0000-00-00 00:00:00','2018-04-30 14:54:35'),
('4','M104','M101','系统角色','/admin/Role/index','903','1','1','fa-hand-stop-o','admin','admin','0000-00-00 00:00:00','2018-04-30 14:54:43'),
('5','M105','M101','系统用户','/admin/User/index','904','1','1','fa-users','admin','admin','0000-00-00 00:00:00','2018-04-30 14:54:51'),
('6','M106','M101','系统日志','/admin/Syslog/index','905','1','1','fa-history','admin','admin','0000-00-00 00:00:00','2018-04-30 14:55:00'),
('7','M107','M101','类型管理','/admin/Type/index','906','0','0','fa-tags','admin','admin','0000-00-00 00:00:00','2018-04-30 14:55:13'),
('8','M108','M101','数据库管理','/admin/Database/index','907','1','1','fa-database','admin','admin','0000-00-00 00:00:00','2018-04-30 14:55:24'),
('9','M109','M101','服务器信息','/admin/Site/server','908','1','1','fa-info-circle','admin','admin','0000-00-00 00:00:00','2018-04-30 14:55:34'),
('10','M1101', 'M101', '图片清理', '/admin/ImageExt/index', '909', '1', '1', 'fa-trash', 'admin', 'admin', '0000-00-00 00:00:00', '2022-09-19 13:44:59'),
('11','M110','0','基础内容','/admin/M110/index','300','1','0','fa-sliders','admin','admin','2017-11-28 11:13:05','2018-04-30 14:48:29'),
('12','M111','M110','站点信息','/admin/Site/index','301','1','1','fa-cog','admin','admin','0000-00-00 00:00:00','2018-04-07 18:45:57'),
('13','M112','M110','公司信息','/admin/Company/index','302','1','1','fa-copyright','admin','admin','0000-00-00 00:00:00','2018-04-07 18:46:09'),
('29','M129','M110','内容栏目','/admin/ContentSort/index','303','1','1','fa-bars','admin','admin','2017-12-26 10:42:40','2018-04-07 18:46:25'),
('30','M130','0','文章内容','/admin/M130/index','400','1','0','fa-file-text-o','admin','admin','2017-12-26 10:45:36','2018-04-30 14:49:47'),
('31','M131','M130','单页内容','/admin/Single/index','401','0','0','fa-file-o','admin','admin','2017-12-26 10:46:35','2018-04-07 18:46:35'),
('32','M132','M130','列表内容','/admin/Content/index','402','0','0','fa-file-text-o','admin','admin','2017-12-26 10:48:17','2018-04-07 21:52:15'),
('36','M136','M156','定制标签','/admin/Label/index','203','1','1','fa-wrench','admin','admin','2018-01-03 11:52:40','2018-04-07 18:44:31'),
('50','M150','M157','留言信息','/admin/Message/index','501','1','1','fa-question-circle-o','admin','admin','2018-02-01 13:20:17','2018-07-07 23:45:09'),
('51','M151','M157','轮播图片','/admin/Slide/index','502','1','1','fa-picture-o','admin','admin','2018-03-01 14:57:41','2018-04-07 18:47:07'),
('52','M152','M157','友情链接','/admin/Link/index','503','1','1','fa-link','admin','admin','2018-03-01 14:58:45','2018-04-07 18:47:16'),
('53','M153','M156','配置参数','/admin/Config/index','201','1','1','fa-sliders','admin','admin','2018-03-21 14:52:05','2018-04-07 18:44:02'),
('61','M1000','M157','文章内链','/admin/Tags/index','505','1','0','fa-random','admin','admin','2019-07-12 08:25:41','2019-07-12 08:26:23'),
('55','M155','M156','模型管理','/admin/Model/index','204','1','1','fa-codepen','admin','admin','2018-03-25 17:16:06','2018-04-07 18:44:40'),
('56','M156','0','全局配置','/admin/M156/index','200','1','0','fa-globe','admin','admin','2018-03-25 17:20:43','2018-04-30 14:43:56'),
('58','M158','M156','模型字段','/admin/ExtField/index','205','1','1','fa-external-link','admin','admin','2018-03-25 21:24:43','2018-04-07 18:44:49'),
('57','M157','0','扩展内容','/admin/M157/index','500','1','0','fa-arrows-alt','admin','admin','2018-03-25 17:27:57','2018-04-30 14:50:34'),
('60','M160','M157','自定义表单','/admin/Form/index','504','1','1','fa-plus-square-o','admin','admin','2018-05-30 18:25:41','2018-05-31 23:55:10'),
('62','M1001','0','会员中心','/admin/M1001/index','600','1','0','fa-user-o','admin','admin','2019-10-04 08:25:41','2019-10-04 08:26:23'),
('63','M1002','M1001','会员等级','/admin/MemberGroup/index','601','1','0','fa-signal','admin','admin','2019-10-04 08:25:41','2019-10-04 08:26:23'),
('64','M1003','M1001','会员字段','/admin/MemberField/index','602','1','0','fa-wpforms','admin','admin','2019-10-04 08:25:41','2019-10-04 08:26:23'),
('65','M1004','M1001','会员管理','/admin/Member/index','603','1','0','fa-users','admin','admin','2019-10-04 08:25:41','2019-10-04 08:26:23'),
('66','M1005','M1001','文章评论','/admin/MemberComment/index','604','1','0','fa-commenting-o','admin','admin','2019-10-04 08:25:41','2019-10-04 08:26:23');

-- --------------------------------------------------------

--
-- 表的结构 `ay_menu_action`
--

DROP TABLE IF EXISTS `ay_menu_action`;
CREATE TABLE `ay_menu_action` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
 `mcode` varchar(20) NOT NULL COMMENT '菜单编码',
 `action` varchar(20) NOT NULL COMMENT '类型编码',
 PRIMARY KEY (`id`),
 KEY `ay_menu_action_mcode` (`mcode`)
) ENGINE=MyISAM AUTO_INCREMENT=79 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_menu_action`
--

INSERT INTO `ay_menu_action` (`id`,`mcode`,`action`) VALUES
('1','M102','mod'),
('2','M102','del'),
('3','M102','add'),
('4','M103','mod'),
('5','M103','del'),
('6','M103','add'),
('7','M104','mod'),
('8','M104','del'),
('9','M104','add'),
('10','M105','mod'),
('11','M105','del'),
('12','M105','add'),
('13','M107','mod'),
('14','M107','del'),
('15','M107','add'),
('16','M111','mod'),
('17','M112','mod'),
('18','M114','mod'),
('19','M114','del'),
('20','M114','add'),
('21','M120','mod'),
('22','M120','del'),
('23','M120','add'),
('24','M129','mod'),
('25','M129','del'),
('26','M129','add'),
('27','M131','mod'),
('28','M132','mod'),
('29','M132','del'),
('30','M132','add'),
('31','M136','mod'),
('32','M136','del'),
('33','M136','add'),
('34','M141','mod'),
('35','M141','del'),
('36','M141','add'),
('37','M142','mod'),
('38','M142','del'),
('39','M142','add'),
('40','M143','mod'),
('41','M143','del'),
('42','M143','add'),
('43','M144','mod'),
('44','M144','del'),
('45','M144','add'),
('46','M145','mod'),
('47','M145','del'),
('48','M145','add'),
('49','M150','del'),
('50','M150','mod'),
('51','M151','mod'),
('52','M151','del'),
('53','M151','add'),
('54','M152','mod'),
('55','M152','del'),
('56','M152','add'),
('57','M155','mod'),
('58','M155','del'),
('59','M155','add'),
('60','M158','mod'),
('61','M158','del'),
('62','M158','add'),
('63','M160','add'),
('64','M160','del'),
('65','M160','mod'),
('66','M1000','add'),
('67','M1000','del'),
('68','M1000','mod'),
('69','M1002','add'),
('70','M1002','del'),
('71','M1002','mod'),
('72','M1003','add'),
('73','M1003','del'),
('74','M1003','mod'),
('75','M1004','add'),
('76','M1004','del'),
('77','M1004','mod'),
('78','M1005','del');

-- --------------------------------------------------------

--
-- 表的结构 `ay_message`
--

DROP TABLE IF EXISTS `ay_message`;
CREATE TABLE `ay_message` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `acode` varchar(20) NOT NULL COMMENT '区域编码',
 `contacts` varchar(10) DEFAULT NULL COMMENT '联系人',
 `mobile` varchar(12) DEFAULT NULL COMMENT '联系电话',
 `content` varchar(500) DEFAULT NULL COMMENT '留言内容',
 `user_ip` varchar(11) NOT NULL DEFAULT '0' COMMENT 'IP地址',
 `user_os` varchar(30) NOT NULL COMMENT '操作系统',
 `user_bs` varchar(30) NOT NULL COMMENT '浏览器',
 `recontent` varchar(500) NOT NULL COMMENT '回复内容',
 `status` char(1) NOT NULL DEFAULT '1' COMMENT '是否前台显示',
 `create_user` varchar(30) NOT NULL COMMENT '创建人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '创建时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 `uid` int(10) unsigned NOT NULL DEFAULT '0',
 PRIMARY KEY (`id`),
 KEY `ay_message_acode` (`acode`)
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_message`
--

-- cleared demo -- cleared demo INSERT INTO `ay_message`;
-- --------------------------------------------------------

--
-- 表的结构 `ay_model`
--

DROP TABLE IF EXISTS `ay_model`;
CREATE TABLE `ay_model` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '序号',
 `mcode` varchar(20) NOT NULL COMMENT '模型编号',
 `name` varchar(50) NOT NULL COMMENT '模型名称',
 `type` char(1) NOT NULL DEFAULT '2' COMMENT '是否列表类型',
 `urlname` varchar(100) NOT NULL DEFAULT '' COMMENT 'URL名称',
 `listtpl` varchar(50) NOT NULL COMMENT '列表页模板',
 `contenttpl` varchar(50) NOT NULL COMMENT '内容页模板',
 `status` char(1) NOT NULL DEFAULT '1' COMMENT '模型状态',
 `issystem` char(1) NOT NULL DEFAULT '0' COMMENT '系统模型',
 `create_user` varchar(30) NOT NULL COMMENT '创建人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '创建时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 PRIMARY KEY (`id`),
 UNIQUE KEY `ay_model_mcode` (`mcode`)
) ENGINE=MyISAM AUTO_INCREMENT=6 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_model`
--

INSERT INTO `ay_model` (`id`,`mcode`,`name`,`type`,`urlname`,`listtpl`,`contenttpl`,`status`,`issystem`,`create_user`,`update_user`,`create_time`,`update_time`) VALUES
('1','1','专题','1','about','','about.html','1','1','admin','admin','2018-04-11 17:16:01','2019-08-05 11:11:44'),
('2','2','新闻','2','list','newslist.html','news.html','1','1','admin','admin','2018-04-11 17:17:16','2019-08-05 11:12:04'),
('3','3','产品','2','list','productlist.html','product.html','1','0','admin','admin','2018-04-11 17:17:46','2019-08-05 11:12:17'),
('4','4','案例','2','list','caselist.html','case.html','1','0','admin','admin','2018-04-11 17:19:53','2019-08-05 11:12:26'),
('5','5','招聘','2','list','joblist.html','job.html','1','0','admin','admin','2018-04-11 17:24:34','2019-08-05 11:12:37');

-- --------------------------------------------------------

--
-- 表的结构 `ay_role`
--

DROP TABLE IF EXISTS `ay_role`;
CREATE TABLE `ay_role` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '角色编号',
 `rcode` varchar(20) NOT NULL COMMENT '角色编码',
 `name` varchar(30) NOT NULL COMMENT '角色名称',
 `description` varchar(50) NOT NULL COMMENT '角色描述',
 `create_user` varchar(30) NOT NULL COMMENT '创建人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '创建时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 PRIMARY KEY (`id`),
 UNIQUE KEY `ay_role_rcode` (`rcode`)
) ENGINE=MyISAM AUTO_INCREMENT=3 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_role`
--

INSERT INTO `ay_role` (`id`,`rcode`,`name`,`description`,`create_user`,`update_user`,`create_time`,`update_time`) VALUES
('1','R101','系统管理员','系统管理员具有所有权限','admin','admin','2017-03-22 11:33:32','2019-08-05 11:22:02'),
('2','R102','内容管理员','内容管理员具有基本内容管理权限','admin','admin','2017-06-01 00:32:02','2019-08-05 11:22:12');

-- --------------------------------------------------------

--
-- 表的结构 `ay_role_area`
--

DROP TABLE IF EXISTS `ay_role_area`;
CREATE TABLE `ay_role_area` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
 `rcode` varchar(20) NOT NULL,
 `acode` varchar(20) NOT NULL,
 PRIMARY KEY (`id`),
 KEY `ay_role_area_rcode` (`rcode`),
 KEY `ay_role_area_acode` (`acode`)
) ENGINE=MyISAM AUTO_INCREMENT=5 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_role_area`
--

INSERT INTO `ay_role_area` (`id`,`rcode`,`acode`) VALUES
('3','R101','cn'),
('4','R102','cn');

-- --------------------------------------------------------

--
-- 表的结构 `ay_role_level`
--

DROP TABLE IF EXISTS `ay_role_level`;
CREATE TABLE `ay_role_level` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `rcode` varchar(20) NOT NULL COMMENT '角色编码',
 `level` varchar(50) NOT NULL COMMENT '权限地址',
 PRIMARY KEY (`id`),
 KEY `ay_role_level_rcode` (`rcode`)
) ENGINE=MyISAM AUTO_INCREMENT=216 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_role_level`
--

INSERT INTO `ay_role_level` (`id`,`rcode`,`level`) VALUES
('165','R101','/admin/Role/index'),
('164','R101','/admin/Menu/mod'),
('163','R101','/admin/Menu/del'),
('162','R101','/admin/Menu/add'),
('161','R101','/admin/Menu/index'),
('160','R101','/admin/Area/mod'),
('159','R101','/admin/Area/del'),
('158','R101','/admin/Area/add'),
('157','R101','/admin/Area/index'),
('156','R101','/admin/M101/index'),
('155','R101','/admin/Tags/mod'),
('154','R101','/admin/Tags/del'),
('153','R101','/admin/Tags/add'),
('152','R101','/admin/Tags/index'),
('151','R101','/admin/Form/mod'),
('150','R101','/admin/Form/del'),
('149','R101','/admin/Form/add'),
('148','R101','/admin/Form/index'),
('147','R101','/admin/Link/mod'),
('146','R101','/admin/Link/del'),
('145','R101','/admin/Link/add'),
('144','R101','/admin/Link/index'),
('143','R101','/admin/Slide/mod'),
('142','R101','/admin/Slide/del'),
('141','R101','/admin/Slide/add'),
('140','R101','/admin/Slide/index'),
('139','R101','/admin/Message/mod'),
('138','R101','/admin/Message/del'),
('137','R101','/admin/Message/index'),
('136','R101','/admin/M157/index'),
('135','R101','/admin/Content/mod'),
('134','R101','/admin/Content/del'),
('133','R101','/admin/Content/add'),
('132','R101','/admin/Content/index'),
('131','R101','/admin/Single/mod'),
('130','R101','/admin/Single/index'),
('129','R101','/admin/M130/index'),
('128','R101','/admin/ContentSort/mod'),
('127','R101','/admin/ContentSort/del'),
('126','R101','/admin/ContentSort/add'),
('125','R101','/admin/ContentSort/index'),
('124','R101','/admin/Company/mod'),
('123','R101','/admin/Company/index'),
('122','R101','/admin/Site/mod'),
('121','R101','/admin/Site/index'),
('120','R101','/admin/M110/index'),
('119','R101','/admin/ExtField/mod'),
('118','R101','/admin/ExtField/del'),
('117','R101','/admin/ExtField/add'),
('116','R101','/admin/ExtField/index'),
('115','R101','/admin/Model/mod'),
('114','R101','/admin/Model/del'),
('113','R101','/admin/Model/add'),
('112','R101','/admin/Model/index'),
('111','R101','/admin/Label/mod'),
('110','R101','/admin/Label/del'),
('109','R101','/admin/Label/add'),
('108','R101','/admin/Label/index'),
('107','R101','/admin/Config/index'),
('106','R101','/admin/M156/index'),
('205','R102','/admin/Link/add'),
('204','R102','/admin/Link/index'),
('203','R102','/admin/Slide/mod'),
('202','R102','/admin/Slide/del'),
('201','R102','/admin/Slide/add'),
('200','R102','/admin/Slide/index'),
('199','R102','/admin/Message/mod'),
('198','R102','/admin/Message/del'),
('197','R102','/admin/Message/index'),
('196','R102','/admin/M157/index'),
('195','R102','/admin/Content/mod'),
('194','R102','/admin/Content/del'),
('193','R102','/admin/Content/add'),
('192','R102','/admin/Content/index'),
('191','R102','/admin/Single/mod'),
('190','R102','/admin/Single/index'),
('189','R102','/admin/M130/index'),
('188','R102','/admin/ContentSort/mod'),
('187','R102','/admin/ContentSort/del'),
('186','R102','/admin/ContentSort/add'),
('185','R102','/admin/ContentSort/index'),
('184','R102','/admin/Company/mod'),
('183','R102','/admin/Company/index'),
('182','R102','/admin/Site/mod'),
('181','R102','/admin/Site/index'),
('180','R102','/admin/M110/index'),
('166','R101','/admin/Role/add'),
('167','R101','/admin/Role/del'),
('168','R101','/admin/Role/mod'),
('169','R101','/admin/User/index'),
('170','R101','/admin/User/add'),
('171','R101','/admin/User/del'),
('172','R101','/admin/User/mod'),
('173','R101','/admin/Syslog/index'),
('174','R101','/admin/Type/index'),
('175','R101','/admin/Type/add'),
('176','R101','/admin/Type/del'),
('177','R101','/admin/Type/mod'),
('178','R101','/admin/Database/index'),
('179','R101','/admin/Site/server'),
('206','R102','/admin/Link/del'),
('207','R102','/admin/Link/mod'),
('208','R102','/admin/Form/index'),
('209','R102','/admin/Form/add'),
('210','R102','/admin/Form/del'),
('211','R102','/admin/Form/mod'),
('212','R102','/admin/Tags/index'),
('213','R102','/admin/Tags/add'),
('214','R102','/admin/Tags/del'),
('215','R102','/admin/Tags/mod');

-- --------------------------------------------------------

--
-- 表的结构 `ay_site`
--

DROP TABLE IF EXISTS `ay_site`;
CREATE TABLE `ay_site` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '站点编号',
 `acode` varchar(20) NOT NULL COMMENT '区域代码',
 `title` varchar(100) NOT NULL COMMENT '站点标题',
 `subtitle` varchar(200) NOT NULL COMMENT '站点副标题',
 `domain` varchar(50) NOT NULL COMMENT '站点地址',
 `logo` varchar(100) NOT NULL COMMENT '站点LOGO地址',
 `keywords` varchar(200) NOT NULL COMMENT '站点关键字',
 `description` varchar(500) NOT NULL COMMENT '站点描述',
 `icp` varchar(30) NOT NULL COMMENT '站点备案',
 `theme` varchar(30) NOT NULL COMMENT '站点主题',
 `statistical` varchar(500) NOT NULL COMMENT '站点统计码',
 `copyright` varchar(200) NOT NULL COMMENT '版权信息',
 PRIMARY KEY (`id`),
 KEY `ay_site_acode` (`acode`)
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_site`
--

INSERT INTO `ay_site` (`id`,`acode`,`title`,`subtitle`,`domain`,`logo`,`keywords`,`description`,`icp`,`theme`,`statistical`,`copyright`) VALUES
('1','cn','PbootCMS-Go','Go 企业网站内容管理系统','','','pbootcms-go,cms','PbootCMS-Go','','default','','Copyright © PbootCMS-Go All Rights Reserved.');

-- --------------------------------------------------------

--
-- 表的结构 `ay_slide`
--

DROP TABLE IF EXISTS `ay_slide`;
CREATE TABLE `ay_slide` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '序号',
 `acode` varchar(20) NOT NULL COMMENT '区域编码',
 `gid` int(10) unsigned NOT NULL COMMENT '分组序号',
 `pic` varchar(100) NOT NULL COMMENT '图片地址',
 `link` varchar(100) NOT NULL COMMENT '跳转链接',
 `title` varchar(50) NOT NULL COMMENT '说明文字',
 `subtitle` varchar(100) NOT NULL COMMENT '副标题/描述',
 `sorting` int(11) NOT NULL COMMENT '排序',
 `create_user` varchar(30) NOT NULL COMMENT '创建人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '创建时间',
 `update_time` datetime NOT NULL COMMENT '修改时间',
 PRIMARY KEY (`id`),
 KEY `ay_slide_acode` (`acode`),
 KEY `ay_slide_gid` (`gid`)
) ENGINE=MyISAM AUTO_INCREMENT=3 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_slide`
--

-- cleared demo -- cleared demo INSERT INTO `ay_slide`;
-- --------------------------------------------------------

--
-- 表的结构 `ay_syslog`
--

DROP TABLE IF EXISTS `ay_syslog`;
CREATE TABLE `ay_syslog` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '日志编号',
 `level` varchar(20) NOT NULL COMMENT '信息等级',
 `event` varchar(200) NOT NULL COMMENT '事件',
 `user_ip` varchar(11) NOT NULL DEFAULT '0' COMMENT '客户端IP',
 `user_os` varchar(30) NOT NULL COMMENT '客户端系统',
 `user_bs` varchar(30) NOT NULL COMMENT '客户端浏览器',
 `create_user` varchar(30) NOT NULL COMMENT '创建人员',
 `create_time` datetime NOT NULL COMMENT '添加时间',
 PRIMARY KEY (`id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- 表的结构 `ay_tags`
--

DROP TABLE IF EXISTS `ay_tags`;
CREATE TABLE `ay_tags` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `acode` varchar(20) NOT NULL COMMENT '区域',
 `name` varchar(50) NOT NULL COMMENT '名称',
 `link` varchar(200) NOT NULL COMMENT '链接',
 `create_user` varchar(30) NOT NULL COMMENT '添加人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新人员',
 `create_time` datetime NOT NULL COMMENT '添加时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 PRIMARY KEY (`id`),
 KEY `ay_tags_acode` (`acode`)
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_tags`
--

-- cleared demo -- cleared demo INSERT INTO `ay_tags`;
-- --------------------------------------------------------

--
-- 表的结构 `ay_type`
--

DROP TABLE IF EXISTS `ay_type`;
CREATE TABLE `ay_type` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '类型编号',
 `tcode` varchar(20) NOT NULL COMMENT '类型编码',
 `name` varchar(30) NOT NULL COMMENT '类型名称',
 `item` varchar(30) NOT NULL COMMENT '类型项',
 `value` varchar(20) NOT NULL DEFAULT '0' COMMENT '类型值',
 `sorting` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '排序',
 `create_user` varchar(30) NOT NULL COMMENT '添加人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新时间',
 `create_time` datetime NOT NULL COMMENT '添加时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 PRIMARY KEY (`id`),
 KEY `ay_type_tcode` (`tcode`)
) ENGINE=MyISAM AUTO_INCREMENT=6 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_type`
--

INSERT INTO `ay_type` (`id`,`tcode`,`name`,`item`,`value`,`sorting`,`create_user`,`update_user`,`create_time`,`update_time`) VALUES
('1','T101','菜单功能','新增','add','1','admin','admin','2017-04-27 07:28:34','2017-08-09 15:25:56'),
('2','T101','菜单功能','删除','del','2','admin','admin','2017-04-27 07:29:08','2017-08-09 15:23:34'),
('3','T101','菜单功能','修改','mod','3','admin','admin','2017-04-27 07:29:34','2017-08-09 15:23:32'),
('4','T101','菜单功能','导出','export','4','admin','admin','2017-04-27 07:30:42','2017-08-09 15:23:29'),
('5','T101','菜单功能','导入','import','5','admin','admin','2017-04-27 07:31:38','2017-08-09 15:23:27');

-- --------------------------------------------------------

--
-- 表的结构 `ay_user`
--

DROP TABLE IF EXISTS `ay_user`;
CREATE TABLE `ay_user` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '用户编号',
 `ucode` varchar(20) NOT NULL COMMENT '用户编码',
 `username` varchar(30) NOT NULL COMMENT '用户账号',
 `realname` varchar(30) NOT NULL COMMENT '真实名字',
 `password` char(32) NOT NULL COMMENT '用户密码',
 `status` char(1) NOT NULL DEFAULT '1' COMMENT '是否启用',
 `login_count` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '登录次数',
 `last_login_ip` varchar(11) NOT NULL DEFAULT '0' COMMENT '最后登录IP',
 `create_user` varchar(30) NOT NULL COMMENT '添加人员',
 `update_user` varchar(30) NOT NULL COMMENT '更新用户',
 `create_time` datetime NOT NULL COMMENT '创建时间',
 `update_time` datetime NOT NULL COMMENT '更新时间',
 PRIMARY KEY (`id`),
 UNIQUE KEY `ay_user_ucode` (`ucode`),
 KEY `ay_user_username` (`username`)
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_user`
--

INSERT INTO `ay_user` (`id`,`ucode`,`username`,`realname`,`password`,`status`,`login_count`,`last_login_ip`,`create_user`,`update_user`,`create_time`,`update_time`) VALUES
('1','10001','admin','超级管理员','14e1b600b1fd579f47433b88e8d85291','1','0','2130706433','admin','admin','2017-05-08 18:50:30','2018-07-17 15:47:27');

-- --------------------------------------------------------

--
-- 表的结构 `ay_user_role`
--

DROP TABLE IF EXISTS `ay_user_role`;
CREATE TABLE `ay_user_role` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '编号',
 `ucode` varchar(20) NOT NULL COMMENT '用户编码',
 `rcode` varchar(20) NOT NULL COMMENT '角色编码',
 PRIMARY KEY (`id`),
 KEY `ay_user_role_ucode` (`ucode`),
 KEY `ay_user_role_rcode` (`rcode`)
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

--
-- 转存表中的数据 `ay_user_role`
--

INSERT INTO `ay_user_role` (`id`,`ucode`,`rcode`) VALUES
('1','10001','R101');

-- --------------------------------------------------------
