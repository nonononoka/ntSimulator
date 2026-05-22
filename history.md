# 5/10

デカ目のリファクタ

https://github.com/nonononoka/ntSimulator/commit/953f312dc306e2dbb0888bb239582a9e971a53ef

もともと一つの`network`パッケージに，`node`インタフェース，`terminalNode`struct，`link`struct，`switch`structが入っていた．

link，switchが，nodeを継承しているので，
まず，nodeインタフェースと，link，switchをそれぞれpackageに分けた．

そうすると，`link`structが，nodeインタフェースと循環参照になってしまった．

これは，`link`structとnodeインタフェースを同じpackageに属させることにした．

nodeインタフェースは，そもそも`link`structからしか使われないので，わざわざpackageを分ける必要はなかったので．

インタフェースに定義させるメソッドは，ポリモーフィズムで使いたいやつだけ定義すれば良い．

以前：

```
type node interface {
    PrintNode()
    NodeId() int
    AddLink(link *Link)
    ReceivePacket(p packetI.PacketI, l *Link)
}
```

`PrintNode`は，node interfaceを受け取って，使っていたわけではないので，
わざわざinterfaceにする必要はない．

一方，`AddLink`とかは，

```
func NewLink(nodeX node, nodeY node, bandwidth float64, delay float64, packetLoss float64, nes *nteventsched.NtEventSched) *Link {
    ...
	nodeX.AddLink(&l)
	nodeY.AddLink(&l)
	return &l
}
```

みたいに，node interfaceを受け取って，使っているのでinterfaceに定義するべき．

# 5/22

arp tableを実装。
このarp tableは本書では、最終到達IPアドレスと、hopしたい相手のMACアドレスを登録するようになっている。
これは現実とは違う。
現実では、hostもrouting tableを持っていて、そこから次に送るルーターのipアドレスを把握して、routerのIPアドレスとrouterのMACアドレスをarp tableから取得する。
が、simulatorの都合上おそらくこうなっている。
