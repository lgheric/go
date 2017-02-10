<?php
//设置默认时区
date_default_timezone_set('Asia/Shanghai');

//echo date('Y-m-d H:i:s',2147443200);
//echo "\n";


$old = array (
  'order_id' =>  2664,
  'scene_id' =>  2575,
  'bidding_start_time' =>  '2017-02-08 21:30:00',
  'bidding_end_time' =>  '0000-00-00 00:00:00',
  'est_elapsed_time' =>  60,
  'act_elapsed_time' =>  0,
  'rank' =>  1,
  'is_timing_order' =>  1

);

$order_list2 = array (
  'order_id' =>  2665,
  'scene_id' =>  2575,
  'bidding_start_time' =>  '2017-02-08 21:30:00',
  'bidding_end_time' =>  '0000-00-00 00:00:00' ,
  'est_elapsed_time' =>  60,
  'act_elapsed_time' =>  0,
  'rank' =>  2,
  'is_timing_order' =>  1

);


print_r(array_merge($order_list2,$old));