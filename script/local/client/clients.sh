tmux new -d -s client0 "./client -rate 100 -payload 250 -port 6100 -time 10 -output client0.log > client0.output 2>&1"
tmux new -d -s client1 "./client -rate 100 -payload 250 -port 6101 -time 10 -output client1.log > client1.output 2>&1"
tmux new -d -s client2 "./client -rate 100 -payload 250 -port 6102 -time 10 -output client2.log > client2.output 2>&1"
tmux new -d -s client3 "./client -rate 100 -payload 250 -port 6103 -time 10 -output client3.log > client3.output 2>&1"