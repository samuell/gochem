/*
 * gonum.go, part of gochem.
 *
 * Copyright 2012 Raul Mera <rmera{at}chemDOThelsinkiDOTfi>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as
 * published by the Free Software Foundation; either version 2.1 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General
 * Public License along with this program.  If not, see
 * <http://www.gnu.org/licenses/>.
 *
 * Gochem is developed at the laboratory for instruction in Swedish, Department of Chemistry,
 * University of Helsinki, Finland.
 *
 */
/***Dedicated to the long life of the Ven. Khenpo Phuntzok Tenzin Rinpoche***/

//Package chem provides atom and molecule structures, facilities for reading and writing some
//files used in computational chemistry and some functions for geometric manipulations and shape
//indicators.

//All the *Vec functions will operate/produce column or row vectors depending on whether the matrix underlying Dense
//is row or column major.

package chem

import (
	"github.com/skelterjohn/go.matrix"
	"math"
	"sort"
)

/*Here I make a -very incomplete- implementation of the gonum api backed by go.matrix, which will enable me to port gochem to gonum.
 * Since the agreement in the gonum community was NOT to build a temporary implementation, I just build the functions that
 * gochem uses, on my own type (which should implement all the relevant gonum interfaces).
 * all the gonum-owned names will start with gn (i.e. RandomFunc becomes gnRandomFunc) so its latter easy to use search and replace to set the
 * correct import path when gonum is implemented (such as gonum.RandomFunc)
 */

//INTERFACES  This part is from GONUM, copyright, the gonum authors.

// Matrix is the basic matrix interface type.
type Matrix interface {
// Dims returns the dimensions of a Matrix.
Dims() (r, c int)

// At returns the value of a matrix element at (r, c). It will panic if r or c are
// out of bounds for the matrix.
At(r, c int) float64
}

type Normer interface {
Norm(o float64) float64
}

type NormerMatrix interface {
	Normer
	Matrix
}











//The main container, must be able to implement any
//gonum interface.
//VecMatrix is a set of vectors in 3D space. The underlying implementation varies.
type VecMatrix struct{
	*Dense
}

//For the pure-go implementation I must implement Dense on top of go.matrix
type Dense struct {
	*matrix.DenseMatrix
}






//Generate and returns a CoorMatrix with arbitrary shape from data.
func NewDense(data []float64, rows, cols int) *Dense {
	if len(data) < cols*rows {
		panic(NotEnoughElements)
	}
	return &Dense{matrix.MakeDenseMatrix(data, rows, cols)}

}



//Returns and empty, but not nil, Dense. It barely allocates memory
func EmptyDense() *Dense {
	var a *matrix.DenseMatrix
	return &Dense{a}

}




//Returns an zero-filled Dense with the given dimensions
//It is to be substituted by the Gonum function.
func gnZeros(rows, cols int) *Dense {
	return &Dense{matrix.Zeros(rows, cols)}
}


//Returns an identity matrix spanning span cols and rows
func gnEye(span int) *Dense {
	A := gnZeros(span, span)
	for i := 0; i < span; i++ {
		A.Set(i, i, 1.0)
	}
	return A
}

func Eye(span int) *Dense {
	return gnEye(span)
}

//Some temporary support function.
//func Eigen(in *Dense, epsilon float64) (*Dense, []float64, error) {
//	i, j, k := gnEigen(in, epsilon)
//	return i, j, k
//}

//This is a facility to sort Eigenvectors/Eigenvalues pairs
//It satisfies the sort.Interface interface.
type eigenpair struct {
	//evecs must have as many rows as evals has elements.
	evecs *VecMatrix
	evals sort.Float64Slice
}

func (E eigenpair) Less(i, j int) bool {
	return E.evals[i] < E.evals[j]
}
func (E eigenpair) Swap(i, j int) {
	E.evals.Swap(i, j)
	//	E.evecs[i],E.evecs[j]=E.evecs[j],E.evecs[i]
	E.evecs.SwapRows(i, j)
}
func (E eigenpair) Len() int {
	return len(E.evals)
}





//gnEigen wraps the matrix.DenseMatrix.Eigen() function in order to guarantee
//That the eigenvectors and eigenvalues are sorted according to the eigenvalues
//It also guarantees orthonormality and handness. I don't know how many of
//these are already guaranteed by Eig(). Will delete the unneeded parts
//And even this whole function when sure.
func gnEigen(in *VecMatrix, epsilon float64) (*VecMatrix, []float64, error) {
	var err error
	if epsilon < 0 {
		epsilon = appzero
	}
	evecsDM, vals, _ := in.Eigen()
	temp := VecMatrix{&Dense{evecsDM}}
	evecs := &temp
	evals := [3]float64{vals.Get(0, 0), vals.Get(1, 1), vals.Get(2, 2)} //go.matrix specific code here.
	f := func() { evecs.T(evecs) }
	if err = gnMaybe(gnPanicker(f)); err != nil {
		return nil, nil, err
	}
	eig := eigenpair{evecs, evals[:]}
	sort.Sort(eig)
	//Here I should orthonormalize vectors if needed instead of just complaining.
	//I think orthonormality is guaranteed by  DenseMatrix.Eig() If it is, Ill delete all this
	//If not I'll add ortonormalization routines.
	eigrows, _ := eig.evecs.Dims()
	vectori := EmptyVecs()
	vectorj := EmptyVecs()
	for i := 0; i < eigrows; i++ {
		vectori.ColView(eig.evecs, i)
		for j := i + 1; j < eigrows; j++ {
			vectorj.ColView(eig.evecs, j)
			if math.Abs(vectori.Dot(vectorj)) > epsilon && i != j {
				return eig.evecs, evals[:], NotOrthogonal
			}
		}
		if math.Abs(vectori.Norm(2)-1) > epsilon {
			//Of course I could just normalize the vectors instead of complaining.
			//err= fmt.Errorf("Vectors not normalized %s",err.Error())

		}
	}
	//Checking and fixing the handness of the matrix.This if-else is Jannes idea,
	//I don't really know whether it works.
	eig.evecs.T(eig.evecs)
	if eig.evecs.Det() < 0 {
		eig.evecs.Scale(-1, eig.evecs) //SSC
	} else {
		/*
			eig.evecs.TransposeInPlace()
			eig.evecs.ScaleRow(0,-1)
			eig.evecs.ScaleRow(2,-1)
			eig.evecs.TransposeInPlace()
		*/
		//	fmt.Println("all good, I guess")
	}
	eig.evecs.T(eig.evecs)
	return eig.evecs, eig.evals, err //Returns a slice of evals
}

//Returns the singular value decomposition of matrix A
func gnSVD(A *Dense) (*Dense, *Dense, *Dense, error) {
	U, s, V, err := A.SVD()
	theU := Dense{U}
	sigma := Dense{s}
	theV := Dense{V}
	return &theU, &sigma, &theV, err

}

//returns a rows,cols matrix filled with gnOnes.
func gnOnes(rows, cols int) *Dense {
	gnOnes := gnZeros(rows, cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			gnOnes.Set(i, j, 1)
		}
	}
	return gnOnes
}

func gnMul(A, B Matrix) *Dense {
	ar, _ := A.Dims()
	_, bc := B.Dims()
	C := gnZeros(ar, bc)
	C.Mul(A, B)
	return C
}

func gnClone(A Matrix) *Dense {
	r, c := A.Dims()
	B := gnZeros(r, c)
	B.Clone(A)
	return B
}

func gnT(A Matrix) *Dense {
	r, c := A.Dims()
	B := gnZeros(c, r)
	B.T(A)
	return B
}

//Methods
/* When gonum is ready, all this functions will take a num.Matrix interface as an argument, instead of a
 * Dense*/

func (F *Dense) Add(A, B Matrix) {
	ar, ac := A.Dims()
	br, bc := B.Dims()
	fr, fc := F.Dims()
	if ac != bc || br != ar || ac != fc || ar != fr {
		panic(gnErrShape)
	}
	for i := 0; i < fr; i++ {
		for j := 0; j < fc; j++ {
			F.Set(i, j, A.At(i, j)+B.At(i, j))
		}
	}

}


func (F *Dense) At(A, B int) float64 {
	return F.Get(A, B)
}

func (F *Dense) Clone(A Matrix) {
	ar, ac := A.Dims()
	fr, fc := F.Dims()
	if ac != fc || ar != fr {
		panic(gnErrShape)
	}

	for i := 0; i < ar; i++ {
		for j := 0; j < ac; j++ {
			F.Set(i, j, A.At(i, j))
		}

	}

}


//Returns an array with the data in the ith row of F
func (F *Dense) Col(a []float64, i int) []float64 {
	r, c := F.Dims()
	if i >= c {
		panic("Matrix: Requested column out of bounds")
	}
	if a==nil{
		a = make([]float64, r, r)
	}
	for j := 0; j < r; j++ {
		if j>=len(a){
			break
		}
		a[j] = F.At(j, i)
	}
	return a
}


func (F *Dense) Dims() (int, int) {
	return F.Rows(), F.Cols()
}

//Dot returns the dot product between 2 vectors or matrices
func (F *Dense) Dot(B Matrix) float64 {
	frows,fcols:=F.Dims()
	brows,bcols:=B.Dims()
	if fcols != bcols || frows != brows {
		panic(gnErrShape)
	}
	a, b := F.Dims()
	A := gnZeros(a, b)
	A.MulElem(F, B)
	return A.Sum()
}

//puts the inverse of B in F or panics if F is non-singular.
//its just a dirty minor adaptation from the code in go.matrix from John Asmuth
//it will be replaced by the gonum implementation when the library is ready.
func (F *Dense) Inv(B Matrix) {
	//fr,fc:=F.Dims()
	ar, ac := B.Dims()
	if ac != ar {
		panic(gnErrSquare)
	}
	var ok bool
	var A *VecMatrix
	A,ok=B.(*VecMatrix)
		if !ok{
			C,ok:=B.(*Dense)
			if !ok{
				panic("Few types are allowed so far")
			}
			A=&VecMatrix{C}

		}
	augt, _ := A.Augment(matrix.Eye(ar))
	aug := &Dense{augt}
	augr, _ := aug.Dims()
	for i := 0; i < augr; i++ {
		j := i
		for k := i; k < augr; k++ {
			if math.Abs(aug.Get(k, i)) > math.Abs(aug.Get(j, i)) {
				j = k
			}
		}
		if j != i {
			aug.SwapRows(i, j)
		}
		if aug.Get(i, i) == 0 {
			panic(gnErrSingular)
		}
		aug.ScaleRow(i, 1.0/aug.Get(i, i))
		for k := 0; k < augr; k++ {
			if k == i {
				continue
			}
			aug.ScaleAddRow(k, i, -aug.Get(k, i))
		}
	}
	F.SubMatrix(aug, 0, ac, ar, ac)
}



//A slightly modified version of John Asmuth's ParalellProduct function.
func (F *Dense) Mul(A, B Matrix) {
	Arows,Acols:=A.Dims()
	Brows,Bcols:=B.Dims()
	if Acols != Brows {
		panic(gnErrShape)
	}
	if F == nil {
		F = gnZeros(Arows, Bcols) //I don't know if the final API will allow this.
	}

	in := make(chan int)
	quit := make(chan bool)

	dotRowCol := func() {
		for {
			select {
			case i := <-in:
				sums := make([]float64, Bcols)
				for k := 0; k < Acols; k++ {
					for j := 0; j < Bcols; j++ {
						sums[j] += A.At(i, k) * B.At(k, j)
					}
				}
				for j := 0; j < Bcols; j++ {
					F.Set(i, j, sums[j])
				}
			case <-quit:
				return
			}
		}
	}

	threads := 2

	for i := 0; i < threads; i++ {
		go dotRowCol()
	}

	for i := 0; i < Arows; i++ {
		in <- i
	}

	for i := 0; i < threads; i++ {
		quit <- true
	}

	return
}

func (F *Dense) MulElem(A, B Matrix) {
	arows, acols := A.Dims()
	brows, bcols := B.Dims()
	frows, fcols := F.Dims()
	if arows != brows || acols != bcols || arows != frows || acols != fcols {
		panic(gnErrShape)
	}
	for i := 0; i < arows; i++ {
		for j := 0; j < acols; j++ {
			F.Set(i, j, A.At(i, j)*B.At(i, j))
		}

	}
}

func (F *Dense) Norm(i float64) float64 {
	//temporary hack
	if i != 2 {
		panic("only 2-norm is implemented")
	}
	return F.TwoNorm()
}



//Returns an array with the data in the ith row of F
func (F *Dense) Row(a []float64, i int) []float64 {
	r, c := F.Dims()
	if i >= r {
		panic("Matrix: Requested row out of bounds")
	}
	if a==nil{
		a = make([]float64, c, c)
	}
	for j := 0; j < c; j++ {
		if j>=len(a){
			break
		}
		a[j] = F.At(i, j)
	}
	return a
}


//Scale the matrix A by a number i, putting the result in the received.
func (F *Dense) Scale(i float64, A Matrix) {
	if A == F { //if A and F points to the same object.
		F.scaleAux(i)
	} else {
		F.Clone(A)
		F.scaleAux(i)
	}
}

func (F *Dense) scaleAux(factor float64) {
	fr, fc := F.Dims()
	for i := 0; i < fr; i++ {
		for j := 0; j < fc; j++ {
			F.Set(i, j, F.At(i, j)*factor)
		}

	}
}


//When go.matrix is abandoned it is necesary to implement SetMatrix
//SetMatrix()
//Copies A into F aligning A(0,0) with F(i,j)
func (F *Dense) SetMatrix(i, j int, A Matrix) {
	fr, fc := F.Dims()
	ar, ac := A.Dims()
	if ar+i > fr || ac+j > fc {
		panic(gnErrShape)
	}
	for l := 0; l < ar; l++ {
		for m := 0; m < ac; m++ {
			F.Set(l+i, m+j, A.At(l, m))
		}
	}
}





//puts in F a matrix consisting in A over B
func (F *Dense) Stack(A, B Matrix) {
	Arows, Acols := A.Dims()
	Brows, Bcols := B.Dims()
	Frows, Fcols := F.Dims()

	if Acols != Bcols || Acols != Fcols || Arows+Brows != Frows {
		panic(gnErrShape)
	}

	for i := 0; i < Arows+Brows; i++ {
		for j := 0; j < Acols; j++ {
			if i < Arows {
				F.Set(i, j, A.At(i, j))
			} else {
				F.Set(i, j, B.At(i-Arows, j))
			}
		}
	}

	return
}

//Subtracts the matrix B from A putting the result in F
func (F *Dense) Sub(A, B Matrix) {
	ar, ac := A.Dims()
	br, bc := B.Dims()
	fr, fc := F.Dims()
	if ac != bc || br != ar || ac != fc || ar != fr {
		panic(gnErrShape)
	}
	for i := 0; i < fr; i++ {
		for j := 0; j < fc; j++ {
			F.Set(i, j, A.At(i, j)-B.At(i, j))
		}
	}

}

//not tested
//returns a copy of the submatrix of A starting by the point i,j and
//spanning rows rows and cols columns.
func (F *Dense) SubMatrix(A *Dense, i, j, rows, cols int) {
	temp := Dense{A.GetMatrix(i, j, rows, cols)}
	F.Clone(&temp)
}



//Sum returns the sum of all elements in matrix A.
func (F *Dense) Sum() float64 {
	Rows, Cols := F.Dims()
	var sum float64
	for i := 0; i < Cols; i++ {
		for j := 0; j < Rows; j++ {
			sum += F.Get(j, i)
		}
	}
	return sum
}

//Transpose
func (F *Dense) T(A Matrix) {
	ar, ac := A.Dims()
	fr, fc := F.Dims()
	if ar != fc || ac != fr {
		panic(gnErrShape)
	}
	var B *Dense
	B,ok:=A.(*Dense)
	if !ok{
		C,ok:=A.(*VecMatrix)
		if !ok{
			panic("Only Dense and VecMatrix are currently accepted")
		}
		B=C.Dense
	}
	//we do it in a different way if you pass the received as the argument
	//(transpose in place) We could use continue for i==j
	if F == B {
/*		for i := 0; i < ar; i++ {
			for j := 0; j < i; j++ {
				tmp := A.At(i, j)
				F.Set(i, j, A.At(j, i))
				F.Set(j, i, tmp)
			}
*/		F.TransposeInPlace()
		}else {
			F.DenseMatrix=B.Transpose()
/*
		for i := 0; i < ar; i++ {
			for j := 0; j < ac; j++ {
				F.Set(j, i, A.At(i, j))
			}
		}
*/
	}
}

//Unit takes a vector and divides it by its norm
//thus obtaining an unitary vector pointing in the same direction as
//vector.
func (F *Dense) Unit(A NormerMatrix) {
	norm := 1.0 / A.Norm(2)
	F.Scale(norm, A)
}



func (F *VecMatrix) View2(A *VecMatrix, i, j, rows, cols int) {
	F.Dense=&Dense{A.GetMatrix(i, j, rows, cols)}
}

/**These are from the current proposal for gonum, by Dan Kortschak. It will be taken out
 * from here when gonum is implemented. The gn prefix is appended to the names to make them
 * unimported and to allow easy use of search/replace to add the "num" prefix when I change to
 * gonum.**/

// A Panicker is a function that may panic.
type gnPanicker func()

// Maybe will recover a panic with a type matrix.Error from fn, and return this error.
// Any other error is re-panicked.
func gnMaybe(fn gnPanicker) (err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(gnError); ok {
				return
			}
			panic(r)
		}
	}()
	fn()
	return
}

// Type Error represents matrix package errors. These errors can be recovered by Maybe wrappers.
type gnError string

func (err gnError) Error() string { return string(err) }

const (
	//RM
	Not3xXMatrix      = gnError("matrix: The other dimmension should be 3")
	NotOrthogonal     = gnError("matrix: Vectors nor orthogonal")
	NotEnoughElements = gnError("matrix: not enough elements")
	//end RM
	gnErrIndexOutOfRange = gnError("matrix: index out of range")
	gnErrZeroLength      = gnError("matrix: zero length in matrix definition")
	gnErrRowLength       = gnError("matrix: row length mismatch")
	gnErrColLength       = gnError("matrix: col length mismatch")
	gnErrSquare          = gnError("matrix: expect square matrix")
	gnErrNormOrder       = gnError("matrix: invalid norm order for matrix")
	gnErrSingular        = gnError("matrix: matrix is singular")
	gnErrShape           = gnError("matrix: dimension mismatch")
	gnErrIllegalStride   = gnError("matrix: illegal stride")
	gnErrPivot           = gnError("matrix: malformed pivot list")
)
