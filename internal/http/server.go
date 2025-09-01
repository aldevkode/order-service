package http

import (
    stdhttp "net/http"
    "encoding/json"
    "strings"

    "github.com/aldevkode/order-service/internal/cache"
)

type Server struct { cache *cache.Store }

func New(c *cache.Store) *Server { return &Server{cache: c} }

func (s *Server) Handler() stdhttp.Handler {
    mux := stdhttp.NewServeMux()

    mux.HandleFunc("/order/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
        id := strings.TrimPrefix(r.URL.Path, "/order/")
        if id == "" { w.WriteHeader(400); w.Write([]byte("missing id")); return }
        if o, ok := s.cache.Get(id); ok {
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(o)
            return
        }
        w.WriteHeader(404); w.Write([]byte("not found"))
    })

    mux.HandleFunc("/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Write([]byte(indexHTML))
    })

    return mux
}

const indexHTML = `<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Order Demo</title>
  <style>body{font-family:system-ui,Segoe UI,Roboto,Arial;margin:40px;max-width:900px} input,button{padding:8px;font-size:16px} .card{border:1px solid #ddd;border-radius:12px;padding:16px;margin-top:16px} pre{white-space:pre-wrap;word-break:break-word}</style>
</head>
<body>
  <h1>Order Demo</h1>
  <p>Введите <code>order_uid</code> и нажмите Поиск.</p>
  <input id="orderId" placeholder="b563feb7b2b84b6test" size="40" />
  <button id="btn">Поиск</button>
  <div id="out" class="card"></div>
  <script>
    async function fetchOrder(id){
      const res = await fetch('/order/'+encodeURIComponent(id));
      const out = document.getElementById('out');
      if(res.ok){
        const data = await res.json();
        out.innerHTML = '<h3>Результат</h3><pre>'+JSON.stringify(data,null,2)+'</pre>';
      } else {
        out.innerHTML = '<span>Не найдено</span>';
      }
    }
    document.getElementById('btn').onclick = ()=>{
      const id = document.getElementById('orderId').value.trim();
      if(id) fetchOrder(id);
    };
  </script>
</body>
</html>`