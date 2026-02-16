// 数据大屏 JavaScript - iOS 风格

let trendChart = null;
let statusChart = null;
let startTime = 0;

// 检测深色模式
const isDarkMode = () => window.matchMedia('(prefers-color-scheme: dark)').matches;

// 主题色
const COLORS = {
    primary: '#885021',
    primaryLight: '#A67C52',
    success: '#34C759',
    warning: '#FF9500',
    danger: '#FF3B30',
    info: '#007AFF',
    textPrimary: () => isDarkMode() ? '#FFFFFF' : '#1C1C1E',
    textSecondary: () => isDarkMode() ? '#98989D' : '#8E8E93',
    border: () => isDarkMode() ? '#38383A' : '#E5E5EA',
    background: () => isDarkMode() ? '#1C1C1E' : '#FFFFFF'
};

// 初始化
document.addEventListener('DOMContentLoaded', function() {
    initCharts();
    loadData();
    // 每 30 秒自动刷新
    setInterval(loadData, 30000);
    // 每秒更新运行时长
    setInterval(updateUptime, 1000);

    // 监听主题变化
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (trendChart) trendChart.dispose();
        if (statusChart) statusChart.dispose();
        initCharts();
        loadData();
    });
});

// 初始化图表
function initCharts() {
    trendChart = echarts.init(document.getElementById('trendChart'));
    statusChart = echarts.init(document.getElementById('statusChart'));

    // 响应式
    window.addEventListener('resize', function() {
        trendChart.resize();
        statusChart.resize();
    });
}

// 加载数据
async function loadData() {
    try {
        const [dashboardRes, trendRes] = await Promise.all([
            fetch('/api/v1/stats/dashboard'),
            fetch('/api/v1/stats/trend?days=7')
        ]);

        const dashboardData = await dashboardRes.json();
        const trendData = await trendRes.json();

        if (dashboardData.code === 200) {
            updateDashboard(dashboardData.data);
        }

        if (trendData.code === 200) {
            updateTrendChart(trendData.data || []);
        }

        // 更新时间
        document.getElementById('lastUpdate').textContent = new Date().toLocaleTimeString('zh-CN', {
            hour: '2-digit',
            minute: '2-digit'
        }) + ' 更新';
    } catch (error) {
        console.error('加载数据失败:', error);
    }
}

// 更新大屏数据
function updateDashboard(data) {
    // 更新卡片 - 带动画效果
    animateNumber('totalRequests', data.totalRequests || 0);
    animateNumber('todayRequests', data.todayRequests || 0);
    animateNumber('uniqueIPs', data.uniqueIPs || 0);
    document.getElementById('avgLatency').textContent = (data.avgLatencyMs || 0).toFixed(0) + 'ms';

    // 保存启动时间
    startTime = data.startTime || 0;

    // 更新图表和列表
    updateStatusChart(data.statusCodeStats || []);
    updateAPIList(data.apiStats || []);
    updateKeywordList(data.topKeywords || []);
}

// 数字动画
function animateNumber(elementId, targetValue) {
    const element = document.getElementById(elementId);
    const currentValue = parseInt(element.textContent.replace(/[^0-9]/g, '')) || 0;
    const diff = targetValue - currentValue;
    const duration = 500;
    const steps = 20;
    const increment = diff / steps;
    let step = 0;

    const timer = setInterval(() => {
        step++;
        const value = Math.round(currentValue + increment * step);
        element.textContent = formatNumber(value);
        if (step >= steps) {
            clearInterval(timer);
            element.textContent = formatNumber(targetValue);
        }
    }, duration / steps);
}

// 更新趋势图
function updateTrendChart(data) {
    const option = {
        tooltip: {
            trigger: 'axis',
            backgroundColor: COLORS.background(),
            borderColor: COLORS.border(),
            borderWidth: 1,
            textStyle: {
                color: COLORS.textPrimary(),
                fontSize: 13
            },
            formatter: function(params) {
                return `<div style="font-weight:600">${params[0].name}</div>
                        <div style="color:${COLORS.primary};margin-top:4px">${params[0].value} 次</div>`;
            }
        },
        grid: {
            left: '3%',
            right: '4%',
            bottom: '3%',
            top: '10%',
            containLabel: true
        },
        xAxis: {
            type: 'category',
            data: data.map(d => {
                const parts = d.date.split('-');
                return `${parts[1]}/${parts[2]}`;
            }),
            axisLine: { show: false },
            axisTick: { show: false },
            axisLabel: {
                color: COLORS.textSecondary(),
                fontSize: 11
            }
        },
        yAxis: {
            type: 'value',
            axisLine: { show: false },
            axisTick: { show: false },
            axisLabel: {
                color: COLORS.textSecondary(),
                fontSize: 11
            },
            splitLine: {
                lineStyle: {
                    color: COLORS.border(),
                    type: 'dashed'
                }
            }
        },
        series: [{
            data: data.map(d => d.count),
            type: 'line',
            smooth: true,
            symbol: 'circle',
            symbolSize: 6,
            areaStyle: {
                color: {
                    type: 'linear',
                    x: 0, y: 0, x2: 0, y2: 1,
                    colorStops: [
                        { offset: 0, color: 'rgba(136, 80, 33, 0.3)' },
                        { offset: 1, color: 'rgba(136, 80, 33, 0.02)' }
                    ]
                }
            },
            lineStyle: {
                color: COLORS.primary,
                width: 3
            },
            itemStyle: {
                color: COLORS.primary,
                borderColor: '#fff',
                borderWidth: 2
            }
        }]
    };
    trendChart.setOption(option);
}

// 更新状态码图表
function updateStatusChart(data) {
    if (!data || data.length === 0) {
        statusChart.setOption({
            graphic: {
                type: 'text',
                left: 'center',
                top: 'center',
                style: {
                    text: '暂无数据',
                    fontSize: 15,
                    fill: COLORS.textSecondary()
                }
            }
        });
        return;
    }

    const colorMap = {
        200: COLORS.success,
        201: '#22D3EE',
        400: COLORS.warning,
        401: '#F97316',
        403: COLORS.danger,
        404: '#A855F7',
        500: '#DC2626'
    };

    const option = {
        tooltip: {
            trigger: 'item',
            backgroundColor: COLORS.background(),
            borderColor: COLORS.border(),
            borderWidth: 1,
            textStyle: {
                color: COLORS.textPrimary(),
                fontSize: 13
            },
            formatter: '{b}: {c} ({d}%)'
        },
        series: [{
            type: 'pie',
            radius: ['45%', '75%'],
            center: ['50%', '50%'],
            avoidLabelOverlap: true,
            itemStyle: {
                borderRadius: 8,
                borderColor: COLORS.background(),
                borderWidth: 3
            },
            label: {
                show: true,
                color: COLORS.textPrimary(),
                fontSize: 12,
                formatter: '{b}'
            },
            labelLine: {
                lineStyle: {
                    color: COLORS.border()
                }
            },
            data: data.map(d => ({
                value: d.count,
                name: d.statusCode,
                itemStyle: { color: colorMap[d.statusCode] || '#6B7280' }
            }))
        }]
    };
    statusChart.setOption(option);
}

// 更新接口列表
function updateAPIList(data) {
    const container = document.getElementById('apiList');
    if (!data || data.length === 0) {
        container.innerHTML = `
            <div class="px-4 py-8 text-center">
                <div class="text-[#8E8E93] text-[15px]">暂无数据</div>
            </div>`;
        return;
    }

    const maxCount = Math.max(...data.map(d => d.count));
    container.innerHTML = data.map((item, index) => `
        <div class="px-4 py-3 flex items-center gap-3">
            <span class="w-6 h-6 flex items-center justify-center rounded-lg text-[11px] font-semibold
                ${index < 3 ? 'bg-primary text-white' : 'bg-[#E5E5EA] dark:bg-[#38383A] text-[#8E8E93]'}">
                ${index + 1}
            </span>
            <div class="flex-1 min-w-0">
                <div class="text-[15px] text-[#1C1C1E] dark:text-white truncate" title="${item.path}">${item.path}</div>
                <div class="w-full bg-[#E5E5EA] dark:bg-[#38383A] rounded-full h-1 mt-1.5">
                    <div class="bg-primary h-1 rounded-full progress-bar" style="width: ${(item.count / maxCount * 100).toFixed(1)}%"></div>
                </div>
            </div>
            <div class="text-right shrink-0">
                <div class="text-[15px] font-semibold text-primary">${formatNumber(item.count)}</div>
                <div class="text-[11px] text-[#8E8E93]">${item.avgLatency.toFixed(0)}ms</div>
            </div>
        </div>
    `).join('');
}

// 更新热词列表
function updateKeywordList(data) {
    const container = document.getElementById('keywordList');
    if (!data || data.length === 0) {
        container.innerHTML = `
            <div class="px-4 py-8 text-center">
                <div class="text-[#8E8E93] text-[15px]">暂无数据</div>
            </div>`;
        return;
    }

    const maxCount = Math.max(...data.map(d => d.searchCount));
    container.innerHTML = data.map((item, index) => `
        <div class="px-4 py-3 flex items-center gap-3">
            <span class="w-6 h-6 flex items-center justify-center rounded-lg text-[11px] font-semibold
                ${index < 3 ? 'bg-info text-white' : 'bg-[#E5E5EA] dark:bg-[#38383A] text-[#8E8E93]'}">
                ${index + 1}
            </span>
            <div class="flex-1 min-w-0">
                <div class="text-[15px] text-[#1C1C1E] dark:text-white truncate" title="${item.keyword}">${item.keyword}</div>
                <div class="w-full bg-[#E5E5EA] dark:bg-[#38383A] rounded-full h-1 mt-1.5">
                    <div class="bg-info h-1 rounded-full progress-bar" style="width: ${(item.searchCount / maxCount * 100).toFixed(1)}%"></div>
                </div>
            </div>
            <div class="text-[15px] font-semibold text-info shrink-0">${formatNumber(item.searchCount)}</div>
        </div>
    `).join('');
}

// 更新运行时长
function updateUptime() {
    if (!startTime) return;

    const now = Math.floor(Date.now() / 1000);
    const diff = now - startTime;

    if (diff < 0) return;

    const days = Math.floor(diff / 86400);
    const hours = Math.floor((diff % 86400) / 3600);
    const minutes = Math.floor((diff % 3600) / 60);
    const seconds = diff % 60;

    let uptime = '';
    if (days > 0) uptime += `${days}天 `;
    uptime += `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;

    document.getElementById('uptime').textContent = uptime;
}

// 数字格式化
function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 10000) {
        return (num / 10000).toFixed(1) + 'W';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}
