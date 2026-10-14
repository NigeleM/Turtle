def mix(a, b)
  (a * 31 + b) % 1000000007
end
best = 1e18; acc = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); acc = 0; 3000000.times { |i| acc = mix(acc, i) }; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{acc} ms=#{'%.1f' % best}"
