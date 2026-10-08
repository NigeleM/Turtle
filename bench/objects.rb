Point = Struct.new(:x, :y)
best = 1e18; total = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); points = Array.new(200000) { |i| Point.new(i, i + 1) }; total = 0; points.each { |p| total += p.x * p.y % 1000 }; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{total} ms=#{'%.1f' % best}"
