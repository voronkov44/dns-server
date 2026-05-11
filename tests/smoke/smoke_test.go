package smoke

import (
	"context"
	"dns-manager/internal/client"
	"dns-manager/internal/dns"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// TestSmokeDNSLifecycle проверяет полный happy-path сценарий работы dns manager
func TestSmokeDNSLifecycle(t *testing.T) {
	apiClient, resolvPath, closeServer := newSmokeClient(t, "")
	defer closeServer()

	ctx := context.Background()

	servers, err := apiClient.ListServers(ctx)
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("initial servers = %v, want empty list", servers)
	}

	servers, err = apiClient.AddServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}
	if !slices.Equal(servers, []string{"8.8.8.8"}) {
		t.Fatalf("servers after add = %v, want %v", servers, []string{"8.8.8.8"})
	}

	servers, err = apiClient.ListServers(ctx)
	if err != nil {
		t.Fatalf("ListServers() after add error = %v", err)
	}
	if !slices.Equal(servers, []string{"8.8.8.8"}) {
		t.Fatalf("servers after list = %v, want %v", servers, []string{"8.8.8.8"})
	}

	servers, err = apiClient.DeleteServer(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("servers after delete = %v, want empty list", servers)
	}

	content := readFile(t, resolvPath)
	if strings.TrimSpace(content) != "" {
		t.Fatalf("resolv.conf content after delete = %q, want empty file", content)
	}
}

// TestSmokeDuplicateAddReturnsConflict проверяет, что повторное добавление
// уже существующего dns-сервера возвращает http 409 conflict
//
// Временный resolv.conf заранее содержит:
//
//	nameserver 8.8.8.8
//
// После попытки добавить 8.8.8.8 ещё раз api должен вернуть ошибку
// Тест проверяет, что ошибка корректно доходит до client.APIError,
// а http-статус равен 409 conflict
func TestSmokeDuplicateAddReturnsConflict(t *testing.T) {
	apiClient, _, closeServer := newSmokeClient(t, "nameserver 8.8.8.8\n")
	defer closeServer()

	_, err := apiClient.AddServer(context.Background(), "8.8.8.8")

	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("AddServer() error = %v, want *client.APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want %d", apiErr.StatusCode, http.StatusConflict)
	}
}

// TestSmokeDeleteMissingServerReturnsNotFound проверяет удаление dns-сервера,
// которого нет в resolv.conf
//
// Временный resolv.conf содержит только:
//
//	nameserver 1.1.1.1
//
// Тест пытается удалить 8.8.8.8
// Ожидаемый результат - HTTP 404 Not Found
//
// Так мы проверяем, что ErrServerNotFound из repository/service
// корректно преобразуется хендлером в http-ответ 404
func TestSmokeDeleteMissingServerReturnsNotFound(t *testing.T) {
	apiClient, _, closeServer := newSmokeClient(t, "nameserver 1.1.1.1\n")
	defer closeServer()

	_, err := apiClient.DeleteServer(context.Background(), "8.8.8.8")

	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("DeleteServer() error = %v, want *client.APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
}

// TestSmokeInvalidIPReturnsBadRequest проверяет валидацию dns-сервера
//
// Тест пытается добавить некорректные значения вместо валидного ip-адреса
// Ожидаемый результат для каждого случая — http 400 Bad Request
//
// Это подтверждает, что service валидирует адрес через net/netip
// и не пропускает невалидные значения в repository и resolv.conf
func TestSmokeInvalidIPReturnsBadRequest(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "not an IP",
			address: "not-an-ip",
		},
		{
			name:    "invalid IPv4 with too many octets",
			address: "8.8.8.8.8",
		},
		{
			name:    "invalid IPv4 octet out of range",
			address: "999.8.8.8",
		},
		{
			name:    "invalid IPv6",
			address: "2001:db8:::1",
		},
		{
			name:    "empty address",
			address: "",
		},
		{
			name:    "blank address",
			address: "   ",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			apiClient, _, closeServer := newSmokeClient(t, "")
			defer closeServer()

			_, err := apiClient.AddServer(context.Background(), test.address)

			var apiErr *client.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("AddServer() error = %v, want *client.APIError", err)
			}
			if apiErr.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
			}
		})
	}
}

// TestSmokeMissingResolvConfIsCreated проверяет сценарий, когда resolv.conf
// ещё несуществует
//
// Тест не создаёт файл заранее, затем вызывает ListServers
// Repository должен автоматически создать временный resolv.conf
// и вернуть пустой список dns-серверов
//
// Тест использует t.TempDir(), поэтому настоящий /etc/resolv.conf
// не читается и не изменяется
func TestSmokeMissingResolvConfIsCreated(t *testing.T) {
	apiClient, resolvPath, closeServer := newSmokeClient(t, "")
	defer closeServer()

	if _, err := os.Stat(resolvPath); !os.IsNotExist(err) {
		t.Fatalf("expected resolv.conf to not exist before request, stat err = %v", err)
	}

	servers, err := apiClient.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("servers = %v, want empty list", servers)
	}

	if _, err := os.Stat(resolvPath); err != nil {
		t.Fatalf("expected repository to create resolv.conf: %v", err)
	}
}

// TestSmokeParallelAddRequestsDoNotOverwriteEachOther проверяет защиту от
// потери данных при параллельных post /dns запросах
//
// Несколько горутин одновременно добавляют разные dns-серверы
// Без mutex в FileRepository возможна race-ситуация:
// два запроса читают старый файл, затем один результат перетирает другой
//
// Ожидаемый результат: все dns-серверы должны оказаться в итоговом списке
// Порядок не проверяется, потому что при параллельном выполнении он не гарантирован
func TestSmokeParallelAddRequestsDoNotOverwriteEachOther(t *testing.T) {
	apiClient, _, closeServer := newSmokeClient(t, "")
	defer closeServer()

	ctx := context.Background()

	addresses := []string{
		"8.8.8.8",
		"1.1.1.1",
		"9.9.9.9",
		"8.8.4.4",
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(addresses))

	for _, address := range addresses {
		wg.Add(1)

		// для каждого адреса запускается отдельная горутина
		go func(address string) {
			defer wg.Done()
			_, err := apiClient.AddServer(ctx, address)
			errCh <- err
		}(address)
	}

	wg.Wait()
	close(errCh)

	// после завершения проверяем, что ошибок нет
	for err := range errCh {
		if err != nil {
			t.Fatalf("parallel AddServer() error = %v", err)
		}
	}

	servers, err := apiClient.ListServers(ctx)
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	// сравниваю как множества, а не через slices.Equal,
	// потому что порядок добавления при параллельных запросах не гарантирован
	if !sameStringSet(servers, addresses) {
		t.Fatalf("servers = %v, want set %v", servers, addresses)
	}
}

// TestSmokeParallelDuplicateAddOnlyCreatesOneEntry проверяет конкурентное
// добавление одного и того же dns-сервера
//
// Несколько горутин одновременно пытаются добавить 8.8.8.8
// Ожидаемый результат:
//   - один запрос успешно добавит dns-сервер;
//   - остальные запросы получат http 409 Conflict;
//   - в итоговом resolv.conf будет только одна запись nameserver 8.8.8.8
//
// Этот тест проверяет одновременно mutex в repository и защиту от дублей
func TestSmokeParallelDuplicateAddOnlyCreatesOneEntry(t *testing.T) {
	apiClient, _, closeServer := newSmokeClient(t, "")
	defer closeServer()

	ctx := context.Background()

	// 5 параллельных запросов пытаются добавить один и тот же адрес
	const address = "8.8.8.8"
	const requests = 5

	var wg sync.WaitGroup
	errCh := make(chan error, requests)

	for i := 0; i < requests; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			_, err := apiClient.AddServer(ctx, address)
			errCh <- err
		}()
	}

	wg.Wait()
	close(errCh)

	// ожидаем:
	// 1 запрос успешный
	//4 запроса получили 409 Conflict

	successCount := 0
	conflictCount := 0

	for err := range errCh {
		if err == nil {
			successCount++
			continue
		}

		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			conflictCount++
			continue
		}

		t.Fatalf("unexpected AddServer() error = %v", err)
	}

	if successCount != 1 {
		t.Fatalf("success count = %d, want 1", successCount)
	}
	if conflictCount != requests-1 {
		t.Fatalf("conflict count = %d, want %d", conflictCount, requests-1)
	}

	servers, err := apiClient.ListServers(ctx)
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if !slices.Equal(servers, []string{address}) {
		t.Fatalf("servers = %v, want %v", servers, []string{address})
	}
}

// TestSmokeParallelDeleteRequestsDoNotCorruptFile проверяет параллельное
// удаление разных dns-серверов
//
// Временный resolv.conf заранее содержит несколько nameserver-записей
// Затем несколько горутин одновременно удаляют разные адреса
//
// Без mutex возможна ситуация, когда один delete перезапишет результат другого,
// и часть удалённых dns-серверов снова появится в файле
//
// Ожидаемый результат: все удаления проходят успешно,
// итоговый список dns-серверов пустой, файл не повреждён
func TestSmokeParallelDeleteRequestsDoNotCorruptFile(t *testing.T) {
	// Начальное состояние файла: четыре dns-сервера,
	// которые будут удаляться параллельно
	initial := strings.Join([]string{
		"nameserver 8.8.8.8",
		"nameserver 1.1.1.1",
		"nameserver 9.9.9.9",
		"nameserver 8.8.4.4",
	}, "\n") + "\n"

	apiClient, _, closeServer := newSmokeClient(t, initial)
	defer closeServer()

	ctx := context.Background()

	addresses := []string{
		"8.8.8.8",
		"1.1.1.1",
		"9.9.9.9",
		"8.8.4.4",
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(addresses))

	// запускаем параллельные delete-запросы для разных адресов
	for _, address := range addresses {
		wg.Add(1)

		go func(address string) {
			defer wg.Done()
			_, err := apiClient.DeleteServer(ctx, address)
			errCh <- err
		}(address)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("parallel DeleteServer() error = %v", err)
		}
	}

	servers, err := apiClient.ListServers(ctx)
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("servers after parallel delete = %v, want empty list", servers)
	}
}

// TestSmokeIPv6DuplicateNormalization проверяет, что ipv6-дубли
// определяются не простым строковым сравнением, а через нормализацию ip
//
// В resolv.conf уже есть:
//
//	nameserver 2001:db8::1
//
// Тест пытается добавить тот же адрес в полной форме:
//
//	2001:0db8:0000:0000:0000:0000:0000:0001
//
// С точки зрения netip.ParseAddr это один и тот же ipv6-адрес
// Ожидаемый результат - http 409 Conflict
//
// Этот тест проверяет, что repository сравнивает адреса через
// netip.ParseAddr(...).String(), а не только как обычные строки
func TestSmokeIPv6DuplicateNormalization(t *testing.T) {
	apiClient, _, closeServer := newSmokeClient(t, "nameserver 2001:db8::1\n")
	defer closeServer()

	_, err := apiClient.AddServer(context.Background(), "2001:0db8:0000:0000:0000:0000:0000:0001")

	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("AddServer() error = %v, want *client.APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want %d", apiErr.StatusCode, http.StatusConflict)
	}
}

// newSmokeClient собирает тестовый экземпляр приложения
//
// Он создаёт временный resolv.conf через t.TempDir(), поднимает реальный
// стек server-side компонентов:
//  1. FileRepository
//  2. Service
//  3. Handler
//  4. httptest.Server
//
// Затем возвращает internal/client.Client, который ходит в этот httptest.Server по http
//
// Благодаря этому smoke-тесты проверяют не отдельную функцию, а полный путь
// Настоящий /etc/resolv.conf в тестах не используется
func newSmokeClient(t *testing.T, resolvContent string) (*client.Client, string, func()) {
	t.Helper()

	// Если resolvContent пустой, файл заранее не создаём
	// Это нужно для теста, который проверяет автоматическое создание resolv.conf
	resolvPath := filepath.Join(t.TempDir(), "resolv.conf")
	if resolvContent != "" {
		if err := os.WriteFile(resolvPath, []byte(resolvContent), 0644); err != nil {
			t.Fatalf("write test resolv.conf: %v", err)
		}
	}

	router := http.NewServeMux()
	repository := dns.NewFileRepository(resolvPath)
	service := dns.NewService(repository)
	dns.NewHandler(router, dns.HandlerDeps{Service: service})

	// httptest.NewServer поднимает реальный http-сервер на случайном свободном порту
	server := httptest.NewServer(router)

	return client.New(server.URL), resolvPath, server.Close
}

// readFile — helper для проверки фактического содержимого временного resolv.conf
// Используется только в smoke-тестах, где нужно убедиться,
// что изменения действительно были записаны в файл
func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}

	return string(content)
}

// sameStringSet сравнивает два string slice как множества,
// то есть без учёта порядка элементов
//
// Это нужно для тестов с параллельными запросами:
// порядок обработки http-запросов не гарантирован,
// но важно убедиться, что все ожидаемые dns-серверы присутствуют
func sameStringSet(left []string, right []string) bool {
	// Сначала считаем количество каждого элемента из left
	// Затем вычитаем элементы из right
	// Если счётчик ушёл ниже нуля, значит наборы отличаются
	if len(left) != len(right) {
		return false
	}

	counts := make(map[string]int, len(left))
	for _, item := range left {
		counts[item]++
	}

	for _, item := range right {
		counts[item]--
		if counts[item] < 0 {
			return false
		}
	}

	return true
}
