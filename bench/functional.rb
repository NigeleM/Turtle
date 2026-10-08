best = 1e18; total = 0; nums = (1..300000).to_a
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); total = nums.map { |n| n * 2 }.select { |n| n % 3 == 0 }.reduce(0) { |a, b| a + b }; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{total} ms=#{'%.1f' % best}"
