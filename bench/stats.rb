best = 1e18; check = 0
3.times do
  t = Process.clock_gettime(Process::CLOCK_MONOTONIC)
  nums = (0...300000).map { |i| (i * 7919 % 1000) / 10.0 }
  n = nums.size
  mean = nums.sum / n
  sd = Math.sqrt(nums.sum { |x| (x - mean)**2 } / (n - 1))
  s = nums.sort
  median = n.odd? ? s[n / 2] : (s[n / 2 - 1] + s[n / 2]) / 2
  pos = 0.9 * (n - 1); lo = pos.floor
  p90 = s[lo] + (pos - lo) * (s[lo + 1] - s[lo])
  check = ((mean + sd + median + p90) * 1000000).round
  best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min
end
puts "result=#{check} ms=#{'%.1f' % best}"
