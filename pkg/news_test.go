package pkg

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func Test_fetchAllNews(t *testing.T) {
	html, err := fetchAllNews(context.Background(), false)
	assert.NoError(t, err)
	os.WriteFile("news.ashx", html, 0644)
}

func Test_parseNewsAndBlogs(t *testing.T) {
	html, err := os.ReadFile("news.ashx")
	assert.NoError(t, err)
	news, blogs, err := parseNewsAndBlogs(html)
	assert.NoError(t, err)
	for _, new_ := range news {
		println(new_.Title)
	}
	for _, blog := range blogs {
		println(blog.Title)
	}
}
