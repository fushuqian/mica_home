/* 开发文档共用脚本：侧栏目录随滚动高亮当前章节。 */
(function () {
  var links = Array.prototype.slice.call(document.querySelectorAll('.toc a'));
  var sections = links
    .map(function (a) { return document.querySelector(a.getAttribute('href')); })
    .filter(Boolean);
  if (!links.length || !sections.length) return;

  function sync() {
    var active = sections[0];
    var y = window.scrollY + 120;
    sections.forEach(function (s) { if (s.offsetTop <= y) active = s; });
    links.forEach(function (a) {
      a.setAttribute('aria-current', String(a.getAttribute('href') === '#' + active.id));
    });
  }

  document.addEventListener('scroll', function () { requestAnimationFrame(sync); }, { passive: true });
  sync();
})();
