// Hello 模块：最小可运行示例。
// 契约：导出 mount(el, api)，返回可选的卸载函数。

let styleInjected = false;

function injectStyle() {
  if (styleInjected) return;
  styleInjected = true;
  const style = document.createElement('style');
  style.id = 'mh-hello-style';
  style.textContent = `
    .mh-hello{display:grid;gap:18px;color:var(--mh-text)}
    .mh-hello .card{
      padding:22px;border-radius:var(--mh-radius);
      background:linear-gradient(150deg,var(--mh-card-a),var(--mh-card-b));
      border:1px solid var(--mh-glass-border);
      box-shadow:inset 0 1px 0 var(--mh-inset),var(--mh-shadow);
    }
    .mh-hello h4{margin:0 0 6px;font:700 17px/1.3 var(--mh-font-d);letter-spacing:-.01em}
    .mh-hello p{margin:0;font-size:13px;line-height:1.7;color:var(--mh-text-2)}
    .mh-hello .metrics{display:grid;grid-template-columns:repeat(auto-fit,minmax(120px,1fr));gap:12px}
    .mh-hello .metric{padding:14px 16px;border-radius:14px;background:var(--mh-soft);border:1px solid var(--mh-line)}
    .mh-hello .metric span{display:block;font:500 10px/1.4 var(--mh-font-m);letter-spacing:.14em;text-transform:uppercase;color:var(--mh-text-3)}
    .mh-hello .metric b{font:700 20px/1.4 var(--mh-font-d);color:var(--teal-deep)}
    .mh-hello .row{display:flex;align-items:center;gap:10px;flex-wrap:wrap;margin-top:14px}
    .mh-hello button{
      border:0;cursor:pointer;padding:10px 18px;border-radius:999px;
      font:600 12.5px/1 inherit;color:var(--mh-bg);background:var(--mh-text);
      transition:transform .3s var(--mh-ease),box-shadow .3s;
    }
    .mh-hello button:hover{transform:translateY(-1px);box-shadow:var(--mh-shadow)}
    .mh-hello button.ghost{background:var(--mh-soft);color:var(--teal-deep);border:1px solid var(--mh-accent-line)}
    .mh-hello code{
      font:500 12px/1.9 var(--mh-font-m);color:var(--teal-deep);
      background:var(--mh-accent-soft);padding:2px 7px;border-radius:6px;
    }
  `;
  document.head.appendChild(style);
}

export function mount(el, api) {
  injectStyle();
  let count = 0;

  el.innerHTML = `
    <div class="mh-hello">
      <div class="card">
        <h4>宿主已经把你的模块装进来了</h4>
        <p>这段界面由模块自己渲染：一个导出 <code>mount(el, api)</code> 的 ES 模块。
           宿主只提供挂载点与受控 API，不关心你用什么框架。</p>
        <div class="row">
          <button data-act="inc">点我 +1</button>
          <button class="ghost" data-act="info">读取模块信息</button>
        </div>
      </div>
      <div class="metrics">
        <div class="metric"><span>模块 ID</span><b>${api.id}</b></div>
        <div class="metric"><span>版本</span><b>v${api.version}</b></div>
        <div class="metric"><span>点击次数</span><b data-count>0</b></div>
        <div class="metric"><span>宿主接口</span><b data-state>待探测</b></div>
      </div>
      <div class="card">
        <h4>模块 API</h4>
        <p><code>api.request(path)</code> 会自动指向本模块的命名空间
           <code>/api/m/${api.id}</code>，模块之间无法越权访问。</p>
      </div>
    </div>`;

  const counter = el.querySelector('[data-count]');
  const state = el.querySelector('[data-state]');

  const onClick = async (e) => {
    const act = e.target.closest('[data-act]')?.dataset.act;
    if (act === 'inc') {
      counter.textContent = String(++count);
      return;
    }
    if (act === 'info') {
      try {
        const info = await api.request('/info');
        state.textContent = '在线';
        api.toast(`${info.name} · 由 ${info.author} 提供`);
      } catch (err) {
        state.textContent = '失败';
        api.toast('模块信息读取失败');
      }
    }
  };
  el.addEventListener('click', onClick);

  return () => el.removeEventListener('click', onClick);
}
