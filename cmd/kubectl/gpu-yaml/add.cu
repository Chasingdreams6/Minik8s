#include <cuda_runtime.h>
#include <iostream>
#include <vector>
#include <fstream>

using namespace std;

// m行N列
__global__ void add_gpu(int *c_matrixA, int *c_matrixB, int *c_matrixC, int n, int m) {
    int ix = threadIdx.x + blockDim.x*blockIdx.x;
	int iy = threadIdx.y + blockDim.y*blockIdx.y;
	unsigned int idx = iy * n + ix;
	if (ix < n && iy < m){
		c_matrixC[idx] = c_matrixA[idx] + c_matrixB[idx];
	}
}

vector<vector<int>> matrix_add(vector<vector<int>> &a, vector<vector<int>> &b) {
    const int m = a.size(), n = a[0].size();
    int *matrixA = (int *)malloc(sizeof(int) * m * n);
    int *matrixB = (int *)malloc(sizeof(int) * m * n);
    int *matrixC = (int *)malloc(sizeof(int) * m * n);

    for (int i = 0; i < m; i++) {
        for (int j = 0; j < n; j++) {
            matrixA[i * n + j] = a[i][j];
            matrixB[i * n + j] = b[i][j];
        }
    }

    int *c_matrixA, *c_matrixB, *c_matrixC;
    cudaMalloc((void **)&c_matrixA, sizeof(int) * n * m);
    cudaMalloc((void **)&c_matrixB, sizeof(int) * n * m);
    cudaMalloc((void **)&c_matrixC, sizeof(int) * n * m);
    cudaMemcpy(c_matrixA, matrixA, sizeof(int) * n * m, cudaMemcpyHostToDevice);
    cudaMemcpy(c_matrixB, matrixB, sizeof(int) * n * m, cudaMemcpyHostToDevice);
    int dimx = 16;
    int dimy = 16;
	dim3 block(dimx, dimy);
    dim3 grid(n / block.x + 1, m / block.y + 1);
    add_gpu<<<grid, block>>>(c_matrixA, c_matrixB, c_matrixC, n, m);
    cudaMemcpy(matrixC, c_matrixC, sizeof(int) * n * m, cudaMemcpyDeviceToHost);
    vector<vector<int>> c(m, vector<int>(n,0));
    for (int i = 0; i < m; i++) {
        for (int j = 0; j < n; j++) {
            c[i][j] = matrixC[i * n + j];
        }
    }
    free(matrixA);
    free(matrixB);
    free(matrixC);
    cudaFree(c_matrixA);
    cudaFree(c_matrixB);
    cudaFree(c_matrixC);
    return c;
}

int main(int argc, char *argv[]) {
    vector<vector<int>> a{{1,2,3},{4,5,6},{7,8,9}}, b{{2,2,2},{2,2,2},{2,2,2}};
    vector<vector<int>> c = matrix_add(a, b);
    for (int i = 0; i < c.size(); i++){
        for (int j = 0; j < c[0].size(); j++){
            std::cout << c[i][j] << " ";
        }
        std::cout  << "\n";
    }

    return 0;
}
