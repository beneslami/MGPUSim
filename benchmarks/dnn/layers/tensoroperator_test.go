package layers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.com/akita/mgpusim/driver"
	"gitlab.com/akita/mgpusim/platform"
)

var _ = Describe("Tensor Operator", func() {
	var (
		gpuDriver *driver.Driver
		context   *driver.Context
		to        *TensorOperator
	)

	BeforeEach(func() {
		_, gpuDriver = platform.MakeEmuBuilder().
			WithISADebugging().
			WithoutProgressBar().
			Build()
		gpuDriver.Run()
		context = gpuDriver.Init()
		to = NewTensorOperator(gpuDriver, context)
	})

	AfterEach(func() {
		gpuDriver.Terminate()
	})

	It("should do gemm", func() {
		a := to.CreateTensor([]int{3, 2})
		to.ToGPU(a, []float32{
			1, 2, 3,
			4, 5, 6,
		})

		b := to.CreateTensor([]int{1, 3})
		to.ToGPU(b, []float32{
			7,
			8,
			9,
		})

		c := to.CreateTensor([]int{1, 2})
		to.ToGPU(c, []float32{
			-1, -2,
		})

		d := to.CreateTensor([]int{1, 2})

		to.Gemm(false, false,
			2, 1, 3,
			0.1, 2.2,
			a, b, c, d)

		outData := make([]float32, 2)
		to.FromGPU(d, outData)

		expectedOutData := []float32{2.8, 7.8}

		for i := range expectedOutData {
			Expect(outData[i]).To(
				BeNumerically("~", expectedOutData[i], 0.001))
		}
	})

	It("should transpose matrix", func() {
		in := to.CreateTensor([]int{2, 3})
		out := to.CreateTensor([]int{3, 2})

		inData := []float32{
			1, 2, 3,
			4, 5, 6,
		}
		to.ToGPU(in, inData)

		to.TransposeMatrix(in, out)

		outData := make([]float32, 3*2)
		to.FromGPU(out, outData)

		Expect(outData).To(Equal([]float32{
			1, 4,
			2, 5,
			3, 6,
		}))
	})

	It("should do general transpose 1", func() {
		in := to.CreateTensor([]int{1, 2, 3, 3})
		in.descriptor = "CNHW"
		inData := []float32{
			15, 18, 9, 30, 36, 18, 25, 30, 15,
			15, 18, 9, 30, 36, 18, 25, 30, 15,
		}
		to.ToGPU(in, inData)

		outData := []float32{
			15, 18, 9, 30, 36, 18, 25, 30, 15,
			15, 18, 9, 30, 36, 18, 25, 30, 15,
		}

		out := to.CreateTensor([]int{2, 1, 3, 3})
		to.TransposeTensor(in, out, []int{1, 0, 2, 3})
		outV := out.Vector()

		for i := range outData {
			Expect(outV[i]).To(BeNumerically("~", outData[i], 1e-3))
		}
		Expect(out.descriptor).To(Equal("NCHW"))
	})

	It("should do general transpose", func() {
		in := to.CreateTensor([]int{2, 4, 3, 3})
		in.descriptor = "CNHW"
		inData := []float32{
			1.111, 1.112, 1.113,
			1.121, 1.122, 1.123,
			1.131, 1.132, 1.133,

			1.211, 1.212, 1.213,
			1.221, 1.222, 1.223,
			1.231, 1.232, 1.233,

			1.311, 1.312, 1.313,
			1.321, 1.322, 1.323,
			1.331, 1.332, 1.333,

			1.411, 1.412, 1.413,
			1.421, 1.422, 1.423,
			1.431, 1.432, 1.433,

			2.111, 2.112, 2.113,
			2.121, 2.122, 2.123,
			2.131, 2.132, 2.133,

			2.211, 2.212, 2.213,
			2.221, 2.222, 2.223,
			2.231, 2.232, 2.233,

			2.311, 2.312, 2.313,
			2.321, 2.322, 2.323,
			2.331, 2.332, 2.333,

			2.411, 2.412, 2.413,
			2.421, 2.422, 2.423,
			2.431, 2.432, 2.433,
		}
		to.ToGPU(in, inData)

		outData := []float32{
			1.111, 1.112, 1.113,
			1.121, 1.122, 1.123,
			1.131, 1.132, 1.133,

			2.111, 2.112, 2.113,
			2.121, 2.122, 2.123,
			2.131, 2.132, 2.133,

			1.211, 1.212, 1.213,
			1.221, 1.222, 1.223,
			1.231, 1.232, 1.233,

			2.211, 2.212, 2.213,
			2.221, 2.222, 2.223,
			2.231, 2.232, 2.233,

			1.311, 1.312, 1.313,
			1.321, 1.322, 1.323,
			1.331, 1.332, 1.333,

			2.311, 2.312, 2.313,
			2.321, 2.322, 2.323,
			2.331, 2.332, 2.333,

			1.411, 1.412, 1.413,
			1.421, 1.422, 1.423,
			1.431, 1.432, 1.433,

			2.411, 2.412, 2.413,
			2.421, 2.422, 2.423,
			2.431, 2.432, 2.433,
		}

		out := to.CreateTensor([]int{4, 2, 3, 3})
		to.TransposeTensor(in, out, []int{1, 0, 2, 3})
		outV := out.Vector()

		for i := range outData {
			Expect(outV[i]).To(BeNumerically("~", outData[i], 1e-3))
		}
		Expect(out.descriptor).To(Equal("NCHW"))
	})

	It("should roate 180", func() {
		in := to.CreateTensor([]int{1, 1, 3, 4})
		in.Init([]float64{
			1, 2, 3, 4,
			5, 6, 7, 8,
			9, 10, 11, 12,
		}, in.size)

		out := to.CreateTensor(in.size)

		to.Rotate180(in, out)

		Expect(out.Vector()).To(Equal([]float64{
			12, 11, 10, 9,
			8, 7, 6, 5,
			4, 3, 2, 1,
		}))
	})

	It("should roate 180, test 2", func() {
		in := to.CreateTensor([]int{2, 3, 3, 4})
		in.Init([]float64{
			1.111, 1.112, 1.113, 1.114,
			1.121, 1.122, 1.123, 1.124,
			1.131, 1.132, 1.133, 1.134,

			1.211, 1.212, 1.213, 1.214,
			1.221, 1.222, 1.223, 1.224,
			1.231, 1.232, 1.233, 1.234,

			1.311, 1.312, 1.313, 1.314,
			1.321, 1.322, 1.323, 1.324,
			1.331, 1.332, 1.333, 1.334,

			2.111, 2.112, 2.113, 2.114,
			2.121, 2.122, 2.123, 2.124,
			2.131, 2.132, 2.133, 2.134,

			2.211, 2.212, 2.213, 2.214,
			2.221, 2.222, 2.223, 2.224,
			2.231, 2.232, 2.233, 2.234,

			2.311, 2.312, 2.313, 2.314,
			2.321, 2.322, 2.323, 2.324,
			2.331, 2.332, 2.333, 2.334,
		}, in.size)

		out := to.CreateTensor(in.size)

		to.Rotate180(in, out)

		goldOut := []float64{
			1.134, 1.133, 1.132, 1.131,
			1.124, 1.123, 1.122, 1.121,
			1.114, 1.113, 1.112, 1.111,

			1.234, 1.233, 1.232, 1.231,
			1.224, 1.223, 1.222, 1.221,
			1.214, 1.213, 1.212, 1.211,

			1.334, 1.333, 1.332, 1.331,
			1.324, 1.323, 1.322, 1.321,
			1.314, 1.313, 1.312, 1.311,

			2.134, 2.133, 2.132, 2.131,
			2.124, 2.123, 2.122, 2.121,
			2.114, 2.113, 2.112, 2.111,

			2.234, 2.233, 2.232, 2.231,
			2.224, 2.223, 2.222, 2.221,
			2.214, 2.213, 2.212, 2.211,

			2.334, 2.333, 2.332, 2.331,
			2.324, 2.323, 2.322, 2.321,
			2.314, 2.313, 2.312, 2.311,
		}

		outV := out.Vector()
		for i := 0; i < len(goldOut); i++ {
			Expect(outV[i]).To(BeNumerically("~", goldOut[i], 1e-3))
		}
	})

	It("should repeat", func() {
		in := to.CreateTensor([]int{9})
		in.Init([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9}, in.size)

		out := to.CreateTensor([]int{36})

		to.Repeat(in, out, 4)

		goldOut := []float64{
			1, 2, 3, 4, 5, 6, 7, 8, 9,
			1, 2, 3, 4, 5, 6, 7, 8, 9,
			1, 2, 3, 4, 5, 6, 7, 8, 9,
			1, 2, 3, 4, 5, 6, 7, 8, 9,
		}

		outV := out.Vector()
		for i := 0; i < len(goldOut); i++ {
			Expect(outV[i]).To(BeNumerically("~", goldOut[i], 1e-3))
		}
	})

	It("should dilate", func() {
		in := to.CreateTensor([]int{1, 1, 3, 3})
		out := to.CreateTensor([]int{1, 1, 7, 5})

		in.Init([]float64{
			1, 2, 3,
			4, 5, 6,
			7, 8, 9,
		}, in.size)

		to.Dilate(in, out, [2]int{3, 2})

		goldOut := []float64{
			1, 0, 2, 0, 3,
			0, 0, 0, 0, 0,
			0, 0, 0, 0, 0,
			4, 0, 5, 0, 6,
			0, 0, 0, 0, 0,
			0, 0, 0, 0, 0,
			7, 0, 8, 0, 9,
		}

		outV := out.Vector()
		for i := 0; i < len(goldOut); i++ {
			Expect(outV[i]).To(BeNumerically("~", goldOut[i], 1e-3))
		}
	})

	It("should do element-wise add", func() {
		in := to.CreateTensor([]int{9})
		in.Init([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9}, in.size)

		out := to.CreateTensor([]int{9})
		out.Init([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9}, in.size)

		to.ElemWiseAdd(out, in, out)

		goldOut := []float64{
			2, 4, 6, 8, 10, 12, 14, 16, 18,
		}

		outV := out.Vector()
		for i := 0; i < len(goldOut); i++ {
			Expect(outV[i]).To(BeNumerically("~", goldOut[i], 1e-3))
		}
	})
})
