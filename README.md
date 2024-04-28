# udptuunel
在tcp不出网的情况下，将tcp流量转化为udp流量传输，并在vps上监听udp流量，然后将该udp流量转回tcp，实现上线msf或cs

useage:

  在vps使用udptotcp
  
  在目标udp出网机器使用tcptoudp



支持:
  msf 非tcp
  
  cs
  
  直接反弹cmd或shell
