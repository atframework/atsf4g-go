<root>
  <include>{{.XRESCONV_XML_PATH}}</include>
  <global>
    <work_dir desc="工作目录，相对于当前xml的目录">.</work_dir>
    <xresloader_path desc="xresloader地址，相对于当前xml的目录">{{.XRESCONV_EXE_PATH}}</xresloader_path>

    <proto desc="协议类型，-p选项">protobuf</proto>
    <output_type desc="输出类型，-t选项，支持多个同时配置多种输出">bin</output_type>
    <!-- <output_type desc="多种输出时可以额外定义某个节点的重命名规则" rename="/(?i)\.bin$/\.json/">json</output_type> -->
    <!-- <output_type desc="可以通过指定class来限制输出的规则" rename="/(?i)\.bin$/\.csv/" class="client" >ue-csv</output_type> -->
    <!-- output_type 里的class标签对应下面item里的class标签，均可配置多个，多个用空格隔开，任意一个class匹配都会启用这个输出 -->
    <proto_file desc="协议描述文件，-f选项">{{.XRESCONV_CONFIG_PB}}</proto_file>

    <output_dir desc="输出目录，-o选项">{{.XRESCONV_BYTES_OUTPUT}}</output_dir>
    <data_src_dir desc="数据源目录，-d选项">{{.XRESCONV_EXECL_SRC}}</data_src_dir>
    <!--<data_version desc="数据版本号，留空则自动生成">1.0.0.0</data_version>-->

    <java_option desc="java选项-最大内存限制2GB">-Xmx2048m</java_option>
    <java_option desc="java选项-客户端模式">-client</java_option>

    <default_scheme name="KeyRow" desc="默认scheme模式参数-Key行号">2</default_scheme>
    <!--<default_scheme name="UeCg-CsvObjectWrapper" desc="Ue-Csv输出的包裹字符">{|}</default_scheme>-->
    <option desc="忽略仅客户端资源">--ignore-field-tags client_only</option>
  </global>
</root>