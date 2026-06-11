package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"big-pickle/internal/handler"
	"big-pickle/internal/model"
	"big-pickle/internal/store"
)

// mockStore implements store.Store for testing
type mockStore struct {
	items map[string]*model.Item
}

func newMockStore() *mockStore {
	return &mockStore{items: make(map[string]*model.Item)}
}

func (m *mockStore) CreateItem(item *model.Item) error {
	m.items[item.ID] = item
	return nil
}

func (m *mockStore) GetItem(id string) (*model.Item, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return item, nil
}

func (m *mockStore) UpdateItem(item *model.Item) error {
	if _, ok := m.items[item.ID]; !ok {
		return store.ErrNotFound
	}
	m.items[item.ID] = item
	return nil
}

func (m *mockStore) DeleteItem(id string) error {
	if _, ok := m.items[id]; !ok {
		return store.ErrNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *mockStore) ListItems() ([]*model.Item, error) {
	list := make([]*model.Item, 0, len(m.items))
	for _, item := range m.items {
		list = append(list, item)
	}
	return list, nil
}

// setupTest creates a test HTTP server with a fresh mock store.
func setupTest(t *testing.T) (*httptest.Server, *mockStore) {
	t.Helper()
	mStore := newMockStore()
	h := handler.NewHandler(mStore)
	srv := httptest.NewServer(h.Router())
	t.Cleanup(srv.Close)
	return srv, mStore
}

// marshalBody is a helper to encode a value as JSON in an io.Reader.
func marshalBody(t *testing.T, v interface{}) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(b)
}

// --------------------------------------------------------------------------
// CREATE
// --------------------------------------------------------------------------

func TestCreateItem(t *testing.T) {
	srv, _ := setupTest(t)

	item := model.Item{
		Name:  "test-item",
		Price: 10.99,
	}
	body := marshalBody(t, item)

	resp, err := http.Post(srv.URL+"/items", "application/json", body)
	if err != nil {
		t.Fatal("POST request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var created model.Item
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal("failed to decode response:", err)
	}

	if created.ID == "" {
		t.Error("expected non-empty ID")
	}
	if created.Name != item.Name {
		t.Errorf("expected Name %q, got %q", item.Name, created.Name)
	}
	if created.Price != item.Price {
		t.Errorf("expected Price %f, got %f", item.Price, created.Price)
	}
}

func TestCreateItemInvalidJSON(t *testing.T) {
	srv, _ := setupTest(t)

	resp, err := http.Post(srv.URL+"/items", "application/json",
		bytes.NewReader([]byte(`{"invalid`)))
	if err != nil {
		t.Fatal("request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// --------------------------------------------------------------------------
// READ
// --------------------------------------------------------------------------

func TestGetItem(t *testing.T) {
	srv, mStore := setupTest(t)

	item := &model.Item{
		ID:    "test-id-1",
		Name:  "test-item",
		Price: 9.99,
	}
	mStore.CreateItem(item)

	resp, err := http.Get(srv.URL + "/items/" + item.ID)
	if err != nil {
		t.Fatal("GET request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var got model.Item
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal("decode failed:", err)
	}

	if got.ID != item.ID {
		t.Errorf("expected ID %q, got %q", item.ID, got.ID)
	}
}

func TestGetItemNotFound(t *testing.T) {
	srv, _ := setupTest(t)

	resp, err := http.Get(srv.URL + "/items/non-existent")
	if err != nil {
		t.Fatal("GET failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestListItems(t *testing.T) {
	srv, mStore := setupTest(t)

	items := []*model.Item{
		{ID: "i1", Name: "Alpha", Price: 1.11},
		{ID: "i2", Name: "Beta", Price: 2.22},
	}
	for _, it := range items {
		mStore.CreateItem(it)
	}

	resp, err := http.Get(srv.URL + "/items")
	if err != nil {
		t.Fatal("GET failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var list []model.Item
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal("decode failed:", err)
	}

	if len(list) != len(items) {
		t.Fatalf("expected %d items, got %d", len(items), len(list))
	}
}

// --------------------------------------------------------------------------
// UPDATE
// --------------------------------------------------------------------------

func TestUpdateItem(t *testing.T) {
	srv, mStore := setupTest(t)

	original := &model.Item{
		ID:    "update-id",
		Name:  "original",
		Price: 5.00,
	}
	mStore.CreateItem(original)

	updated := model.Item{
		ID:    original.ID,
		Name:  "updated",
		Price: 7.50,
	}
	body := marshalBody(t, updated)

	req, err := http.NewRequest(http.MethodPut, srv.URL+"/items/"+original.ID, body)
	if err != nil {
		t.Fatal("PUT request creation failed:", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("PUT request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result model.Item
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal("decode failed:", err)
	}

	if result.Name != updated.Name {
		t.Errorf("expected Name %q, got %q", updated.Name, result.Name)
	}
	if result.Price != updated.Price {
		t.Errorf("expected Price %f, got %f", updated.Price, result.Price)
	}
}

func TestUpdateItemNotFound(t *testing.T) {
	srv, _ := setupTest(t)

	item := model.Item{ID: "nonexistent", Name: "nope", Price: 0}
	body := marshalBody(t, item)

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/items/nonexistent", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("PUT failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// --------------------------------------------------------------------------
// DELETE
// --------------------------------------------------------------------------

func TestDeleteItem(t *testing.T) {
	srv, mStore := setupTest(t)

	item := &model.Item{
		ID:    "delete-me",
		Name:  "going-away",
		Price: 12.34,
	}
	mStore.CreateItem(item)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/items/"+item.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("DELETE request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify it is gone
	_, err = mStore.GetItem(item.ID)
	if err != store.ErrNotFound {
		t.Error("item was not removed from store")
	}
}

func TestDeleteItemNotFound(t *testing.T) {
	srv, _ := setupTest(t)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/items/ghost", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("DELETE failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}