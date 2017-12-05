<?php
$path = './check';
$files = scandir($path);
 foreach ($files as $v){
  	if($v != '.' && $v != '..'){
  		$result = substr($v,0,strrpos($v, '.'));
  		$result = trim($result);
  		$arr[] = $result;	
  	}
}
// print_r($result); die;
$arr1 = $arr;
$new_arr = array();
foreach($arr1 as $key=>$value){
	$key1 = iconv('gb2312','utf-8',$key);
	$value1 = iconv('gb2312','utf-8',$value);
	$new_arr[$key1] = $value1;
}
$arr2 = ['王瑞臣','刘亚鑫','牛江涛','艾磊','李楠','陈钊',
'马志远','刘鹏飞','张涛','张佳佳','刘宴昌','贲红宝','王浩宇',
'张豪杰','陈静','刘真','杨燕霞','张明','周光辉','王新宁','刘银辉',
'宋恩来','邸国梁','王少聪','司薇薇','刘林方','谢志杰','刘学宽','谢子琰',
'葛鑫鹏','牟金吉','宋宜蒙','刘东升','孙晓慧','班思雨'];
$end = array_diff($arr2,$new_arr);
$end2 = implode("/",$end);
echo date('Y-m-d').'<br>';
echo "<h2>未交人:<h2>";
echo "<h3><font color='red'>$end2</font></h3>";



