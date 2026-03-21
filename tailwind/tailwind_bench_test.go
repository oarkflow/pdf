package tailwind

import "testing"

func BenchmarkParseSingleClass(b *testing.B) {
	p := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse("bg-blue-500")
	}
}

func BenchmarkParseMultipleClasses(b *testing.B) {
	p := New()
	classes := "flex items-center justify-between p-4 m-2 bg-blue-500 text-white rounded-lg shadow-md border border-gray-200 w-full max-w-lg mx-auto text-center font-bold text-xl"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse(classes)
	}
}

func BenchmarkParseResponsiveVariants(b *testing.B) {
	p := New()
	classes := "sm:flex sm:items-center md:grid md:grid-cols-3 md:gap-4 lg:text-xl lg:p-8 sm:bg-red-500 md:bg-green-500 lg:bg-blue-500 sm:w-full md:w-1/2 lg:w-1/3"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse(classes)
	}
}
