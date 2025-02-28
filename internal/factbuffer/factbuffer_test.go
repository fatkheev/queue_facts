package factbuffer

import (
    "io"
    "net/http"
    "net/http/httptest"
    "net/url"
    "sync/atomic"
    "testing"
    "time"
)

// Искуственный RoundTripper. Заменем схему и хост на адрес тестового сервера.
type rewritingTransport struct {
    base *url.URL
    rt   http.RoundTripper
}

// Переписываем схему и хост запроса
func (rt *rewritingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    // Устанавливаем схему и хост из тестового сервера
    req.URL.Scheme = rt.base.Scheme
    req.URL.Host = rt.base.Host
    return rt.rt.RoundTrip(req)
}

// Проверяем, что факты успешно отправляются
// Тестовый сервер всегда отвечает 200
// и увеличиваем счетчик, чтобы убедиться, что все факты отправлены
func TestFactBuffer_Success(t *testing.T) {
    var reqCount int32

    // Создаем тестовый сервер.
    // Читаем запрос и возвращаем 200
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        atomic.AddInt32(&reqCount, 1) // Увеличиваем счетчик
        // Читаем тело запроса и имитировать обработку
        _, err := io.ReadAll(r.Body)
        if err != nil {
            t.Logf("Ошибка чтения тела запроса: %v", err)
        }
        r.Body.Close()
        w.WriteHeader(http.StatusOK)
    }))
    defer ts.Close()

    // Парсим URL тестового сервера, чтобы использовать его в нашем кастомном транспорте
    testURL, err := url.Parse(ts.URL)
    if err != nil {
        t.Fatalf("Ошибка парсинга URL тестового сервера: %v", err)
    }

    fb := NewFactBuffer(10)
    // Заменяем транспорт клиента и перенаправляем запросы на наш тестовый сервер
    fb.client.Transport = &rewritingTransport{
        base: testURL,
        rt:   http.DefaultTransport,
    }

    for i := 1; i <= 5; i++ {
        fact := Fact{
            PeriodStart:         "2024-12-01",
            PeriodEnd:           "2024-12-31",
            PeriodKey:           "month",
            IndicatorToMoID:     "227373",
            IndicatorToMoFactID: "0",
            Value:               "value",
            FactTime:            "2024-12-31",
            IsPlan:              "0",
            AuthUserID:          "40",
            Comment:             "buffer fatkheev",
        }
        fb.AddFact(fact)
    }

    fb.Stop()

    // Проверяем, что тестовый сервер получил ровно 5 запросов
    if reqCount != 5 {
        t.Errorf("Ожидалось, что сервер получит 5 запросов, получено %d", reqCount)
    }
}

// Проверяет, что факт, который находится в очереди более 5 минут,
// не отправляется на сервер. Эмулируем ситуацию, когда факт был добавлен 6 минут назад.
func TestFactBuffer_Expired(t *testing.T) {
    var reqCount int32

    // Тестовый сервер, который всегда отвечает 200
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        atomic.AddInt32(&reqCount, 1)
        _, err := io.ReadAll(r.Body)
        if err != nil {
            t.Logf("Ошибка чтения тела запроса: %v", err)
        }
        r.Body.Close()
        w.WriteHeader(http.StatusOK)
    }))
    defer ts.Close()

    testURL, err := url.Parse(ts.URL)
    if err != nil {
        t.Fatalf("Ошибка парсинга URL тестового сервера: %v", err)
    }

    fb := NewFactBuffer(10)
    fb.client.Transport = &rewritingTransport{
        base: testURL,
        rt:   http.DefaultTransport,
    }

    // Создаем факт, кторый типа хранится 6 минут
    fact := Fact{
        PeriodStart:         "2024-12-01",
        PeriodEnd:           "2024-12-31",
        PeriodKey:           "month",
        IndicatorToMoID:     "227373",
        IndicatorToMoFactID: "0",
        Value:               "expired",
        FactTime:            "2024-12-31",
        IsPlan:              "0",
        AuthUserID:          "40",
        Comment:             "buffer fatkheev",
        EnqueuedAt:          time.Now().Add(-6 * time.Minute),
    }
    // Добавляем факт напрямую в очередь без функции AddFact, чтобы не менять время поступления
    fb.queue <- fact

    fb.Stop()

    // Ожидаем, что сервер не получит ни одного запроса, факт устарел
    if reqCount != 0 {
        t.Errorf("Ожидалось, что сервер получит 0 запросов для устаревшего факта, получено %d", reqCount)
    }
}
